package main

import (
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/drone/drone-go/drone"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/oauth2"
)

type status string

const (
	success status = "success"
	failure status = "failure"
	unknown status = "unknown"
)

type repoEntry struct {
	namespace string
	name      string
	counter   int64
	status    status
}

var (
	l *log.Logger = log.New(os.Stdout, "[drone-exporter] ", 2)

	// Metrics
	metricGaugesMap  map[string]*prometheus.GaugeVec
	apiCounterMetric *prometheus.CounterVec
	metricPrefix     = getEnv("METRICS_PREFIX", "drone_exporter")

	// Service Setup
	httpPort        = getEnv("HTTP_PORT", "9100")
	httpPath        = getEnv("HTTP_PATH", "metrics")
	intervalMinutes = getEnvInt("INTERVAL_MINUTES", 60)

	// Drone Config
	droneClient drone.Client
	droneURL    = getEnv("URL", "https://cloud.drone.io/")
	droneApikey = getEnv("API_KEY", "")

	// Filters
	namespaceFilter = getEnv("NAMESPACE", "")
	repoFilter      = getEnv("REPO", "")

	// Repo Data
	reposData map[int64]repoEntry = make(map[int64]repoEntry)
)

// getEnv Get an env vairable and set a default
func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv("DRONE_EXPORTER_" + key); ok {
		return value
	}
	return fallback
}

// getEnvInt Get an env vairable and set a default with integer type
func getEnvInt(key string, fallback int) int {
	if value, ok := os.LookupEnv("DRONE_EXPORTER_" + key); ok {
		value, err := strconv.Atoi(value)
		if err != nil {
			l.Printf("Error converting interval: %s", err.Error())
			os.Exit(1)
		}
		return value
	}
	return fallback
}

func init() {
	// Register metric gauges
	metricGaugesMap = map[string]*prometheus.GaugeVec{
		"build status": prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metricPrefix,
			Name:      "build_status",
			Help:      "The current build status of the repo",
		}, []string{"namespace", "name"}),

		"build count": prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metricPrefix,
			Name:      "build_count",
			Help:      "The number of builds of the repo",
		}, []string{"namespace", "name"}),
	}

	for _, metric := range metricGaugesMap {
		prometheus.MustRegister(metric)
	}

	// Register api count metric
	apiCounterMetric = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricPrefix,
			Name:      "api_request_count",
			Help:      "Number of API requests to the drone api",
		}, []string{"url"},
	)
	prometheus.MustRegister(apiCounterMetric)

	// Setup drone api client
	config := new(oauth2.Config)
	droneClient = drone.NewClient(droneURL, config.Client(
		oauth2.NoContext,
		&oauth2.Token{
			AccessToken: droneApikey,
		},
	))
}

// Increment counter for the number of API requests
func incAPICount() {
	apiCounterMetric.WithLabelValues(droneURL).Inc()
}

// Refresh the repo build data
func refreshRepoList() (map[int64]repoEntry, error) {
	/*
		Hits the drone API to get the repo list for the authenticated user account
		uses the build count to monitor for additional build information.
	*/
	repos, err := droneClient.RepoList()
	incAPICount()
	if err != nil {
		return nil, err
	}

	namespaceRegexp := regexp.MustCompile(namespaceFilter)
	repoRegexp := regexp.MustCompile(repoFilter)

	for _, repo := range repos {
		if repo.Active && (namespaceFilter == "" || namespaceRegexp.MatchString(repo.Namespace)) && (repoFilter == "" || repoRegexp.MatchString(repo.Name)) {
			if _, ok := reposData[repo.ID]; ok {
				if repo.Counter != reposData[repo.ID].counter || reposData[repo.ID].status == unknown {
					// Build data should be refreshed, change in build counter. (Also update the counter value)
					reposData[repo.ID] = repoEntry{
						namespace: repo.Namespace,
						name:      repo.Name,
						counter:   repo.Counter,
						status:    fetchBuildStatus(repo.Namespace, repo.Name, repo.Branch),
					}
				}
			} else if repo.Counter > 0 {
				// Disregard any repos that do not yet have a build status
				// Initial setup for repo data
				reposData[repo.ID] = repoEntry{
					namespace: repo.Namespace,
					name:      repo.Name,
					counter:   repo.Counter,
					status:    fetchBuildStatus(repo.Namespace, repo.Name, repo.Branch),
				}
			}
		}
	}

	return reposData, nil
}

// Fetch the build status for a given repo
func fetchBuildStatus(namespace, name, branch string) status {
	builds, err := droneClient.BuildList(namespace, name, drone.ListOptions{})
	incAPICount()
	if err != nil {
		return unknown
	}

	for _, build := range builds {
		if build.Event == "push" && build.Source == branch && build.Target == branch {
			if build.Status == "success" {
				return success
			} else if build.Status == "failure" {
				return failure
			}
		}
	}

	return unknown
}

func main() {
	go func() {
		for {
			l.Println("Refreshing metrics")
			repos, err := refreshRepoList()
			if err != nil {
				l.Println("Failed to fetch the repo list:", err)
			}

			// Set the metrics for each repo in the repo list
			for _, repo := range repos {
				// Set the metric for the number of builds
				metricGaugesMap["build count"].WithLabelValues(repo.namespace, repo.name).Set(float64(repo.counter))

				// Set metric for repo status
				if repo.status == success {
					metricGaugesMap["build status"].WithLabelValues(repo.namespace, repo.name).Set(1)
				} else if repo.status == failure {
					metricGaugesMap["build status"].WithLabelValues(repo.namespace, repo.name).Set(0)
				}
			}

			time.Sleep(time.Duration(intervalMinutes) * time.Minute)
		}
	}()

	l.Printf("Listening for HTTP requests at: 0.0.0.0:%v\n", httpPort)
	http.Handle("/"+httpPath, promhttp.Handler())
	http.ListenAndServe(":"+httpPort, nil)
}

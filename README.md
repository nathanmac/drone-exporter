# Drone CI Metrics

[![Build Status](https://cloud.drone.io/api/badges/nathanmac/drone-exporter/status.svg)](https://cloud.drone.io/nathanmac/drone-exporter)

Monitor your drone builds and output to prometheus. This is done by calling the Drone API on a regular interval and updating the prometheus metrics.

These metrics include:

- Build Status
- Build Count
- Total API Count

## Usage

### Docker

We have a Docker image for you to use on docker hub. By default you'll need to expose port `9100`. Metric would then be available at `/metrics`

```
nathanmac/drone-exporter
```

### Environment Variables

You can customise the service using a number of environment variables.

| Name                              | Default                    | Description                                                                               |
|-----------------------------------|----------------------------|-------------------------------------------------------------------------------------------|
| `DRONE_EXPORTER_HTTP_PORT`        | `9100`                     | The HTTP port that you will need to expose if running in docker                           |
| `DRONE_EXPORTER_HTTP_PATH`        | `/metrics`                 | The HTTP endpoint to to expose prometheus metrics on                                      |
| `DRONE_EXPORTER_URL`              | `https://cloud.drone.io/`  | The URL of the drone service, defaults to the cloud version of drone.io                   |
| `DRONE_EXPORTER_API_KEY`          | `Required`                 | The API Key for accessing the drone service                                               |
| `DRONE_EXPORTER_NAMESPACE`        |                            | Filter by organisation name, supports regex (optional, leave empty for all organisations) |
| `DRONE_EXPORTER_REPO`             |                            | Filter by the name of the repo, supports regex (optional, leave empty for all repos)      |
| `DRONE_EXPORTER_INTERVAL_MINUTES` | `60`                       | The interval used to refresh the drone metrics                                            |
| `DRONE_EXPORTER_METRICS_PREFIX`   | `drone_exporter`           | The prefix for the prometheus metrics names                                               |
| `DRONE_EXPORTER_NAME`             | `default`                  | A name to put into prometheus metric keys to help identify, if using multiple instances   |

## Prometheus output

This is an example of the output you get from the service.

```
# HELP drone_exporter_api_count Number of API requests to the drone api
# TYPE drone_exporter_api_count counter
drone_exporter_api_count 2

# HELP drone_exporter_build_count The number of builds of the repo
# TYPE drone_exporter_build_count gauge
drone_exporter_build_count{name="drone-exporter",namespace="nathanmac"} 42

# HELP drone_exporter_build_status The current build status of the repo
# TYPE drone_exporter_build_status gauge
drone_exporter_build_status{name="drone-exporter",namespace="nathanmac"} 1
```

## Grafana Dashboard

TODO - Add link to grafana dashboard download and example.

## Known issues

- This exporter does not support paginated back through the build list therefore if a build to the primary branch is not 

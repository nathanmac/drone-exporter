FROM alpine

# Sort out CA certs
RUN apk --update add ca-certificates

# Needed for go binary to run properly
RUN mkdir /lib64 && ln -s /lib/libc.musl-x86_64.so.1 /lib64/ld-linux-x86-64.so.2

WORKDIR /

COPY dist/drone-exporter /usr/bin/drone-exporter

ENTRYPOINT ["/usr/bin/drone-exporter"]

# syntax=docker/dockerfile:1

FROM golang:1.26.4@sha256:f96cc555eb8db430159a3aa6797cd5bae561945b7b0fe7d0e284c63a3b291609 AS build
WORKDIR /app
ARG BUILD_VERSION=dev

# Download Go modules
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code. Note the slash at the end, as explained in
# https://docs.docker.com/reference/dockerfile/#copy
COPY *.go ./
COPY internal/ internal/

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-X 'main.Version=${BUILD_VERSION}'" -o /turbostat-exporter

# sid (unstable) is used deliberately: stable/testing ship a linux-cpupower
# package with an older turbostat that doesn't recognize newer CPU
# generations. This trades reproducible package versions for turbostat
# currency; the base image itself is still pinned by digest below.
FROM debian:sid-slim@sha256:02aca65ed93bc0a93e4cd1f04cda43bee12daf09283ddff0fe8d03713c16e966

RUN <<EOF
    set -eu
    apt-get update
    DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends linux-cpupower
    apt-get clean
    rm -rf /var/lib/apt/lists/*
EOF

COPY --from=build /turbostat-exporter /usr/bin/turbostat-exporter

CMD [ "/usr/bin/turbostat-exporter" ]

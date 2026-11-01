# syntax=docker/dockerfile:1

FROM golang:1.27.3@sha256:2920bb124f1a2e12a1d39f0cab756ae0068441ee1da7655dec7f5e374944095b AS build
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
FROM debian:sid-slim@sha256:17d1843dae9ca66f1617c1e464f40d06059ccefb0c606e8b54687f253af1684e

RUN <<EOF
    set -eu
    apt-get update
    DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends linux-cpupower
    apt-get clean
    rm -rf /var/lib/apt/lists/*
EOF

COPY --from=build /turbostat-exporter /usr/bin/turbostat-exporter

CMD [ "/usr/bin/turbostat-exporter" ]

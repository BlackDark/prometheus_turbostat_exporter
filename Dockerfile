# syntax=docker/dockerfile:1

FROM golang:1.26.5@sha256:705e964a93a2fd2e75c7d59bb7d781b57e30f12293ffde5175c69229e18fb678 AS build
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
FROM debian:sid-slim@sha256:76b6251aaac0ebb6aca1afbc780717aab7e455038c07cb0cb23facb33d241c7d

RUN <<EOF
    set -eu
    apt-get update
    DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends linux-cpupower
    apt-get clean
    rm -rf /var/lib/apt/lists/*
EOF

COPY --from=build /turbostat-exporter /usr/bin/turbostat-exporter

CMD [ "/usr/bin/turbostat-exporter" ]

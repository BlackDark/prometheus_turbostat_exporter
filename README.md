# Prometheus Turbostat Exporter

This Go application is a Prometheus exporter for turbostat metrics. 
It collects CPU and core statistics using the `turbostat` tool and exposes them in a format that Prometheus can scrape.

## Dashboard

You can use the provided dashboard in the folder `dashboards` or use this shared ones: https://grafana.com/grafana/dashboards/23537

## Features

- **Prometheus Integration**: Exposes metrics via an HTTP server for Prometheus to scrape.
- **Dynamic Metric Registration**: Automatically registers metrics based on turbostat output headers.
- **Configuration via Environment Variables**: Customize behavior using `.env` files.
- **Background Collection Mode**: Optionally collect metrics in the background at specified intervals.

## How to use

You can download the binaries for available platforms in the [Releases](https://github.com/BlackDark/prometheus_turbostat_exporter/releases).

- Run with `turbostat-exporter`. Default listener on `0.0.0.0:9101` (also displayed as logs),
- or run with docker (but must be run as priviliged to have all information available):
  `docker run -p 9101:9101 --privileged ghcr.io/blackdark/prometheus_turbostat_exporter:main`
- or deploy to Kubernetes with the [Helm chart](./helm/turbostat-exporter) (see below).

## Kubernetes / Helm

A Helm chart is available at [`helm/turbostat-exporter`](./helm/turbostat-exporter),
published as an OCI artifact:

```bash
helm install turbostat-exporter oci://ghcr.io/blackdark/charts/turbostat-exporter --version <chart-version>
```

It deploys turbostat-exporter as a **DaemonSet** (one pod per node, since
turbostat reads host hardware counters rather than per-container state).

Unlike the plain-Docker instructions above, the chart does **not** use
`--privileged`. turbostat needs to read `/dev/cpu/*/msr`, which on a stock
kernel is `root:root` mode `0600`, and the kernel's msr driver additionally
requires the `CAP_SYS_RAWIO` capability regardless of UID - so by default the
chart runs the container as root with only `SYS_RAWIO`/`SYS_ADMIN` added to
an otherwise fully-dropped capability set (`privileged: false`,
`readOnlyRootFilesystem: true`, seccomp `RuntimeDefault`). It also creates
**no Kubernetes RBAC** (no Role/ClusterRole) - the app never talks to the
Kubernetes API. See the [chart README](./helm/turbostat-exporter/README.md#why-this-chart-needs-elevated-permissions)
for the full rationale and for how to reduce this further if your hosts
already restrict msr access differently.

## Example scrape output

Part of the output from the scrape:

```txt
...
# HELP turbostat_cores 
# TYPE turbostat_cores gauge
turbostat_cores{core="0",package="0",type="avg_mhz"} 23
turbostat_cores{core="0",package="0",type="bzy_mhz"} 1228
turbostat_cores{core="0",package="0",type="c1"} 2
turbostat_cores{core="0",package="0",type="c1e"} 275
turbostat_cores{core="0",package="0",type="c3"} 0
...
```

## Installation

1. **Clone the repository**:
   ```bash
   git clone <repository-url>
   cd <repository-directory>
   ```

2. **Install dependencies**:
   Ensure Go is installed on your system, then run:
   ```bash
   go mod tidy
   ```

3. **Build the application**:
   ```bash
   go build -o turbostat_exporter
   ```

## Usage

1. **Run the exporter**:
   ```bash
   ./turbostat_exporter
   ```

2. **Access metrics**:
   Open a browser or use `curl` to access `http://localhost:9101/metrics`.

## Configuration

The application can be configured using environment variables defined in a `.env` file:

- `TURBOSTAT_EXPORTER_LOG_LEVEL`: Set logging level (`debug` or `info`).
- `TURBOSTAT_EXPORTER_DEFAULT_COLLECT_SECONDS`: Default interval for data collection.
- `TURBOSTAT_EXPORTER_DEBUG_CAT_EXEC`: If set to `true`, uses a test mode with sample data.
- `TURBOSTAT_COLLECT_IN_BACKGROUND`: Enables background data collection if set to `true`.
- `TURBOSTAT_COLLECT_IN_BACKGROUND_INTERVAL`: Interval for background data collection.
- `TURBOSTAT_LISTEN_ADDR`: Address/port the HTTP server listens on (default `0.0.0.0:9101`).
- `TURBOSTAT_BASIC_AUTH_ENABLED`: Enable HTTP basic auth on `/metrics` if set to `true`.
- `TURBOSTAT_BASIC_AUTH_USERNAME` / `TURBOSTAT_BASIC_AUTH_PASSWORD`: Required when basic auth is enabled.

## Development

To modify the code:

1. **Edit source files** in your preferred editor.
2. **Rebuild the application** using the build command above.

CI runs [golangci-lint](https://golangci-lint.run/) (config in `.golangci.yml`)
and [zizmor](https://docs.zizmor.sh/) (GitHub Actions workflow security
scanning) on every push and pull request. Run them locally with:

```bash
golangci-lint run ./...
zizmor .github/workflows/
```

## Releasing

Two independent things can be released from this repository:

- **The application**: push a tag matching `v*.*.*` (e.g. `v1.2.3`). This
  triggers `.github/workflows/build.yml` (binaries attached to a GitHub
  Release) and `.github/workflows/docker.yml` (multi-tag image pushed to
  `ghcr.io/blackdark/prometheus_turbostat_exporter`).
- **The Helm chart**: bump `version` in
  [`helm/turbostat-exporter/Chart.yaml`](./helm/turbostat-exporter/Chart.yaml),
  then push a tag matching `helm-vX.Y.Z` matching that version (e.g. bump to
  `0.2.0` and tag `helm-v0.2.0`). This triggers
  `.github/workflows/helm.yml`, which refuses to run if the tag and
  `Chart.yaml` version don't match, then packages the chart, pushes it to
  `oci://ghcr.io/blackdark/charts/turbostat-exporter`, and attaches the
  packaged `.tgz` to a GitHub Release.

The chart's version is intentionally independent from the application's
version (see `appVersion` in `Chart.yaml`, which just documents the exporter
version the chart defaults to) - a chart-only fix doesn't force an
application release, and vice versa.

## Dependencies

- [Prometheus Client Golang](https://github.com/prometheus/client_golang)
- [Logrus](https://github.com/sirupsen/logrus)
- [Godotenv](https://github.com/joho/godotenv)

# Prometheus Turbostat Exporter

This Go application is a Prometheus exporter for turbostat metrics. 
It collects CPU and core statistics using the `turbostat` tool and exposes them in a format that Prometheus can scrape.

## Features

- **Prometheus Integration**: Exposes metrics via an HTTP server for Prometheus to scrape.
- **Dynamic Metric Registration**: Automatically registers metrics based on turbostat output headers.
- **Configuration via Environment Variables**: Customize behavior using `.env` files.
- **Background Collection Mode**: Optionally collect metrics in the background at specified intervals.

## How to use

You can download the binaries for available platforms in the [Releases](https://github.com/BlackDark/prometheus_turbotstat_exporter/releases).

## Example scrape output

Part of the output from the scrape:

```txt
...
turbostat_cpu_states_percent{num_cpu="6",type="c1e"} 0.03
turbostat_cpu_states_percent{num_cpu="6",type="c3"} 0.05
turbostat_cpu_states_percent{num_cpu="6",type="c6"} 1.37
turbostat_cpu_states_percent{num_cpu="6",type="c7s"} 0
turbostat_cpu_states_percent{num_cpu="6",type="c8"} 1.11
turbostat_cpu_states_percent{num_cpu="6",type="c9"} 0.13
turbostat_cpu_states_percent{num_cpu="6",type="poll"} 0
turbostat_cpu_states_percent{num_cpu="7",type="c1"} 0.01
turbostat_cpu_states_percent{num_cpu="7",type="c10"} 96.91
turbostat_cpu_states_percent{num_cpu="7",type="c1e"} 0.03
turbostat_cpu_states_percent{num_cpu="7",type="c3"} 0.01
turbostat_cpu_states_percent{num_cpu="7",type="c6"} 1.36
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

## Development

To modify the code:

1. **Edit source files** in your preferred editor.
2. **Rebuild the application** using the build command above.

## Dependencies

- [Prometheus Client Golang](https://github.com/prometheus/client_golang)
- [Logrus](https://github.com/sirupsen/logrus)
- [Godotenv](https://github.com/joho/godotenv)

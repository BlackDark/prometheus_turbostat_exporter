package main

import (
	"blackdark/turbostat-exporter/internal"
	"bytes"
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	Version                                = "development"
	defaultSleepTimer        time.Duration = 5
	isCommandCat                           = false
	isBackgroundMode                       = false
	backgroundCollectSeconds               = 30
	basicAuthUsername        string
	basicAuthPassword        string
	basicAuthEnabled         = false
	listenAddr               = "0.0.0.0:9101"
)

func main() {
	flagVersion := flag.Bool("version", false, "prints the version")
	flag.Parse()
	if *flagVersion {
		fmt.Println(Version)
		os.Exit(0)
	}

	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	parser := internal.NewTurbostatParser()
	exporter := internal.NewTurbostatExporter()

	updateFunc := createUpdateFunc(parser, exporter)

	startServer(context.TODO(), updateFunc)
}

func createUpdateFunc(parser *internal.TurbostatParser, exporter *internal.TurbostatExporter) func(time.Duration) {
	return func(collectionTimeSecnds time.Duration) {
		content, err := executeProgram(0)
		if err != nil {
			log.Error().Msgf("Failed to run turbostat: %v", err)
			os.Exit(1)
		}

		headers, rows, err := parseTurbostatOutput(content)
		if err != nil {
			log.Error().Msgf("Failed to parse turbostat output: %v", err)
			os.Exit(1)
		}

		log.Info().Msgf("Found %d headers, %d data lines", len(headers), len(rows))
		log.Debug().Msgf("Headers: %s", headers)

		parser.SetupColumnParsers(headers)

		parsedRows, err := parser.ParseRows(rows)
		if err != nil {
			log.Error().Msgf("Failed to parse turbostat data: %v", err)
			os.Exit(1)
		}

		exporter.Update(parsedRows)
	}
}

func startServer(ctx context.Context, updateFunc func(time.Duration)) {
	fmt.Println("Prometheus turbostat exporter - created by BlackDark (https://github.com/BlackDark/prometheus_turbostat_exporter)")
	parseConfiguration()

	updateFunc(0)

	if isBackgroundMode {
		log.Debug().Msgf("Starting ticker")
		ticker := time.NewTicker(time.Duration(backgroundCollectSeconds) * time.Second)

		go func() {
			updateFunc(defaultSleepTimer)
			for {
				select {
				case <-ticker.C:
					log.Debug().Msgf("Ticker update")
					updateFunc(defaultSleepTimer)
				case <-ctx.Done():
					log.Debug().Msgf("Stop background updater")
					ticker.Stop()
					return
				}
			}
		}()
	}

	var metricsHandler http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {
		if !isBackgroundMode {
			updateFunc(defaultSleepTimer)
		}
		promhttp.Handler().ServeHTTP(w, r)
	}

	if basicAuthEnabled {
		metricsHandler = internal.BasicAuth(metricsHandler, basicAuthUsername, basicAuthPassword)
	}

	http.Handle("/metrics", metricsHandler)
	log.Info().Msgf("Starting server on %s", listenAddr)
	log.Fatal().Err(http.ListenAndServe(listenAddr, nil)).Msg("")
}

func parseTurbostatOutput(raw string) ([]string, [][]string, error) {
	var rows [][]string

	r := csv.NewReader(strings.NewReader(raw))
	r.FieldsPerRecord = -1 // disable row length checks as they have different lengths
	r.Comma = '\t'

	headers, err := r.Read()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse headers: %w", err)
	}

	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}

		if err != nil {
			return nil, nil, fmt.Errorf("failed to parse column: %w", err)
		}

		rows = append(rows, row)
	}

	return headers, rows, nil
}

func executeProgram(collectTimeSeconds int) (string, error) {
	var cmd *exec.Cmd

	if isCommandCat {
		content, err := os.ReadFile("data/sandy-bridge.tsv")
		if err != nil {
			return "", err
		}
		return string(content), nil
	} else {
		cmd = exec.Command("turbostat", "--quiet", "sleep", strconv.Itoa(collectTimeSeconds))
	}
	log.Trace().Msgf("Executing command: %s", cmd.Args)

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	err := cmd.Run()
	if err != nil {
		log.Fatal().Msgf("Failed to run turbostat: %v", err)
		return "", err
	}

	return out.String(), nil
}

func parseConfiguration() {
	godotenv.Load()

	if val, ok := os.LookupEnv("TURBOSTAT_EXPORTER_LOG_LEVEL"); ok {
		level, err := zerolog.ParseLevel(val)
		if err != nil {
			fmt.Printf("Invalid log level provided %s. Setting default to INFO", val)
		} else {
			zerolog.SetGlobalLevel(level)
		}
	}

	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	// use the default if not set
	if val, ok := os.LookupEnv("TURBOSTAT_EXPORTER_DEFAULT_COLLECT_SECONDS"); ok {
		if convertVal, err := strconv.Atoi(val); err == nil {
			defaultSleepTimer = time.Duration(convertVal)
		}
	}

	log.Info().Msgf("Configured turbostat collecting time of %d seconds", defaultSleepTimer)

	if val, ok := os.LookupEnv("TURBOSTAT_EXPORTER_DEBUG_CAT_EXEC"); ok {
		if val == "true" {
			isCommandCat = true
			log.Info().Msgf("Running in testing 'cat' mode. Will not execute turbostat.")
		}
	}

	if val, ok := os.LookupEnv("TURBOSTAT_COLLECT_IN_BACKGROUND"); ok {
		if val == "true" {
			isBackgroundMode = true
		}
	}

	if val, ok := os.LookupEnv("TURBOSTAT_COLLECT_IN_BACKGROUND_INTERVAL"); ok {
		if convertVal, err := strconv.Atoi(val); err == nil {
			backgroundCollectSeconds = convertVal
		}
	}

	if isBackgroundMode {
		log.Info().Msgf("Running collector in background with interval %d.", backgroundCollectSeconds)
	} else {
		log.Info().Msgf("Running collector in active mode (on request will execute turbostat)")
	}

	// use the default if not set
	if val, ok := os.LookupEnv("TURBOSTAT_BASIC_AUTH_ENABLED"); ok {
		if convertVal, err := strconv.ParseBool(val); err == nil {
			if convertVal {
				basicAuthEnabled = true
			}
		}
	}

	if basicAuthEnabled {
		if val, ok := os.LookupEnv("TURBOSTAT_BASIC_AUTH_USERNAME"); ok {
			basicAuthUsername = val
		} else {
			log.Fatal().Msg("BasicAuth enabled but could not read username.")
			panic("Please correct your config.")
		}

		if val, ok := os.LookupEnv("TURBOSTAT_BASIC_AUTH_PASSWORD"); ok {
			basicAuthPassword = val
		} else {
			log.Fatal().Msg("BasicAuth enabled but could not read password.")
			panic("Please correct your config.")
		}

		log.Info().Msg("Enabled basic auth")
	}

	if val, ok := os.LookupEnv("TURBOSTAT_LISTEN_ADDR"); ok {
		listenAddr = val
	}
}

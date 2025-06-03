package main

import (
	"blackdark/turbostat-exporter/internal"
	"context"
	"flag"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
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

	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	parser := internal.NewTurbostatParser()
	exporter := internal.NewTurbostatExporter()

	updateFunc := createUpdateFunc(parser, exporter)

	startServer(context.TODO(), updateFunc)
}

func createUpdateFunc(parser *internal.TurbostatParser, exporter *internal.TurbostatExporter) func(time.Duration) {
	return func(collectionTimeSecnds time.Duration) {
		content, err := executeProgram(int(collectionTimeSecnds.Seconds()))
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to run turbostat")
		}

		headers, rows, err := internal.ParseTurbostatOutput(content)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to parse turbostat output")
		}

		log.Debug().Msgf("Found %d headers, %d data lines", len(headers), len(rows))
		log.Debug().Msgf("Headers: %s", headers)

		parsedRows := parser.ParseRowsSimple(headers, rows)

		extractedCategories := "Categories found - "
		// Debug: print how many rows are in each category
		for _, cat := range []string{"package", "core", "cpu", "total"} {
			catRows := parsedRows[cat]
			extractedCategories += fmt.Sprintf("%s: %d, ", cat, len(catRows))
		}

		log.Debug().Msgf("%s", extractedCategories)

		// Collect all rows from all categories
		allRows := make([]internal.TurbostatRow, 0)
		for _, v := range parsedRows {
			for _, r := range v {
				allRows = append(allRows, *r)
			}
		}
		exporter.Update(allRows)
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
			updateFunc(defaultSleepTimer * time.Second)
			for {
				select {
				case <-ticker.C:
					log.Debug().Msgf("Ticker update")
					updateFunc(defaultSleepTimer * time.Second)
				case <-ctx.Done():
					log.Debug().Msgf("Stop background updater")
					ticker.Stop()
					return
				}
			}
		}()
	}

	metricsHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !isBackgroundMode {
			updateFunc(defaultSleepTimer * time.Second)
		}
		promhttp.Handler().ServeHTTP(w, r)
	})

	if basicAuthEnabled {
		metricsHandler = internal.BasicAuth(metricsHandler, basicAuthUsername, basicAuthPassword)
	}

	http.Handle("/metrics", metricsHandler)
	log.Info().Msgf("Starting server on %s", listenAddr)
	log.Fatal().Err(http.ListenAndServe(listenAddr, nil)).Msg("")
}

func executeProgram(collectTimeSeconds int) (string, error) {
	if isCommandCat {
		content, err := os.ReadFile("data/prox.tsv")
		if err != nil {
			return "", err
		}
		return string(content), nil
	}

	// Create a temp file for turbostat output
	dir := os.TempDir()
	tmpFile := filepath.Join(dir, fmt.Sprintf("turbostat_%d_%d.tsv", os.Getpid(), rand.Intn(1000000)))

	turbostatCmd := fmt.Sprintf("turbostat --quiet sleep %d > %s", collectTimeSeconds, tmpFile)
	cmd := exec.Command("/bin/sh", "-c", turbostatCmd)
	log.Trace().Msgf("Executing command: %s", turbostatCmd)

	if err := cmd.Start(); err != nil {
		log.Fatal().Err(err).Msg("Failed to start turbostat")
		return "", err
	}

	// Sleep in Go for the measurement duration plus a small buffer
	time.Sleep(time.Duration(collectTimeSeconds)*time.Second + 500*time.Millisecond)

	// Wait for the turbostat process to finish
	err := cmd.Wait()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to run turbostat")
		return "", err
	}

	// Read the output from the temp file
	content, err := os.ReadFile(tmpFile)
	if err != nil {
		return "", err
	}
	// Clean up the temp file
	_ = os.Remove(tmpFile)

	return string(content), nil
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

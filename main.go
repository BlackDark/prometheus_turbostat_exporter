package main

import (
	"bytes"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"maps"
	"math"
	"net/http"
	"os"
	"os/exec"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	//log "github.com/sirupsen/logrus"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type TurbostatType string

var TurbostatTypes = newTurbostatTypeReg()

func newTurbostatTypeReg() *turbostatTypReg {
	return &turbostatTypReg{
		Total: "total",
		Core:  "core",
		Cpu:   "cpu",
	}
}

type turbostatTypReg struct {
	Total TurbostatType
	Core  TurbostatType
	Cpu   TurbostatType
}

type metricMapping struct {
	metric     prometheus.Metric
	index      string
	coreIndex  int
	metricType TurbostatType
}

func buildMetricList(reader io.Reader) []metricMapping {
	headers, data := parseOutput(reader)
	headersLen := len(headers)
	dataLen := len(data)

	log.Info().Msgf("Found %d headers, %d data lines", headersLen, dataLen)
	log.Debug().Msgf("Headers: %s", headers)

	coreIndices := []int{}
	cpuIndices := []int{}
	s_union := map[int]int{}

	for i := 1; i < dataLen; i++ {
		coreValue := int(data[i]["Core"].(float64))
		if _, ok := s_union[coreValue]; !ok {
			s_union[coreValue] = i
		}
		cpuIndices = append(cpuIndices, i)
	}

	for _, v := range s_union {
		coreIndices = append(coreIndices, v)
	}

	log.Info().Msgf("Total cores %d, Total cpus (threads) %d", len(coreIndices), len(cpuIndices))

	pollIndex := slices.Index(headers, "POLL")
	firstCpuPercent := slices.IndexFunc(headers, func(n string) bool {
		return strings.HasPrefix(n, "CPU%")
	})
	coreTmpIndex := slices.Index(headers, "CoreTmp")
	firstPkgState := slices.Index(headers, "CPUGFX%") + 1
	lastPkgState := slices.Index(headers, "CPU%LPI") - 1

	m := make(map[string]prometheus.Metric)
	listOfMetrics := []metricMapping{}

	headerTotalIndices := []int{}
	headerCoreIndices := []int{}
	headerCpuIndices := []int{}
	headerPkgIndices := []int{}

	// cpu = one thread, core = physical core
	for i := 2; i < headersLen; i++ {
		if (i < pollIndex) || (i >= coreTmpIndex && i < firstPkgState) || (i > lastPkgState && i < headersLen) {
			headerTotalIndices = append(headerTotalIndices, i)
		}

		if i >= pollIndex && i < firstCpuPercent {
			headerCpuIndices = append(headerCpuIndices, i)
		}

		if i >= firstCpuPercent && i < coreTmpIndex {
			headerCoreIndices = append(headerCoreIndices, i)
		}

		if i >= firstPkgState && i <= lastPkgState {
			headerPkgIndices = append(headerPkgIndices, i)
		}
	}

	log.Debug().Msgf("Extracted following header indices for states: Total %d, Core %d, CPU %d, Pkg %d", headerTotalIndices, headerCoreIndices, headerCpuIndices, headerPkgIndices)

	registerer := promauto.With(prometheus.WrapRegistererWithPrefix("turbostat_", prometheus.DefaultRegisterer))

	for _, i := range headerTotalIndices {
		val := headers[i]
		pkgLabels := make(prometheus.Labels)
		var metricName = "total"

		convertedVal := strings.ToLower(val)
		isPercentage := strings.Index(convertedVal, "%")

		if isPercentage > -1 {
			convertedVal = strings.ReplaceAll(convertedVal, "%", "")
			metricName += "_percent"
		}

		pkgLabels["type"] = convertedVal

		metric := registerer.NewGauge(prometheus.GaugeOpts{
			Name:        metricName,
			Help:        "Total value for something",
			ConstLabels: pkgLabels,
		})

		m[fmt.Sprintf("total_%s", val)] = metric

		listOfMetrics = append(listOfMetrics, metricMapping{
			metric:     metric,
			index:      val,
			metricType: TurbostatTypes.Total,
		})
	}

	for _, i := range headerCoreIndices {
		val := headers[i]

		pkgLabels := make(prometheus.Labels)
		var metricName = "core_states"

		convertedVal := strings.ToLower(val)
		isPercentage := strings.Index(convertedVal, "%")

		var labelTypeValue string = convertedVal

		if isPercentage > -1 {
			splitted := strings.Split(convertedVal, "%")

			if splitted[1] != "" {
				labelTypeValue = splitted[1]
			} else {
				labelTypeValue = splitted[0]
			}

			metricName += "_percent"
		}

		pkgLabels["type"] = labelTypeValue

		metric := registerer.NewGauge(prometheus.GaugeOpts{
			Name:        fmt.Sprintf("total_%s", metricName),
			Help:        "Total value for something",
			ConstLabels: pkgLabels,
		})

		m[fmt.Sprintf("total_%s", val)] = metric

		listOfMetrics = append(listOfMetrics, metricMapping{
			metric:     metric,
			index:      val,
			metricType: TurbostatTypes.Total,
		})

		for _, coreV := range coreIndices {
			coreLabels := map[string]string{
				"num_core": prependStringWithZeros(strconv.Itoa(int(data[coreV]["CPU"].(float64))), len(coreIndices)),
			}
			maps.Copy(coreLabels, pkgLabels)

			listOfMetrics = append(listOfMetrics, metricMapping{
				metric: registerer.NewGauge(prometheus.GaugeOpts{
					Name:        metricName,
					Help:        "Single value for something",
					ConstLabels: coreLabels,
				}),
				index:      val,
				coreIndex:  coreV,
				metricType: TurbostatTypes.Core,
			})
		}
	}

	for _, i := range headerCpuIndices {
		val := headers[i]

		pkgLabels := make(prometheus.Labels)
		var metricName = "cpu_states"

		convertedVal := strings.ToLower(val)
		isPercentage := strings.Index(convertedVal, "%")

		var labelTypeValue string = convertedVal

		if isPercentage > -1 {
			splitted := strings.Split(convertedVal, "%")

			if splitted[1] != "" {
				labelTypeValue = splitted[1]
			} else {
				labelTypeValue = splitted[0]
			}

			metricName += "_percent"
		}

		pkgLabels["type"] = labelTypeValue

		metric := registerer.NewGauge(prometheus.GaugeOpts{
			Name:        fmt.Sprintf("total_%s", metricName),
			Help:        "Total value for something",
			ConstLabels: pkgLabels,
		})

		m[fmt.Sprintf("total_%s", val)] = metric

		listOfMetrics = append(listOfMetrics, metricMapping{
			metric:     metric,
			index:      val,
			metricType: TurbostatTypes.Total,
		})

		for _, cpuV := range cpuIndices {
			coreLabels := map[string]string{
				"num_cpu": prependStringWithZeros(strconv.Itoa(int(data[cpuV]["CPU"].(float64))), len(cpuIndices)),
			}
			maps.Copy(coreLabels, pkgLabels)

			listOfMetrics = append(listOfMetrics, metricMapping{
				metric: registerer.NewGauge(prometheus.GaugeOpts{
					Name:        metricName,
					Help:        "Single value for something",
					ConstLabels: coreLabels,
				}),
				index:      val,
				coreIndex:  cpuV,
				metricType: TurbostatTypes.Cpu,
			})
		}
	}

	for _, i := range headerPkgIndices {
		val := headers[i]

		pkgLabels := make(prometheus.Labels)
		var metricName = "pkg_states"

		convertedVal := strings.ToLower(val)
		isPercentage := strings.Index(convertedVal, "%")

		var labelTypeValue string = convertedVal

		if isPercentage > -1 {
			splitted := strings.Split(convertedVal, "%")

			if splitted[1] != "" {
				labelTypeValue = splitted[1]
			} else {
				labelTypeValue = splitted[0]
			}

			metricName += "_percent"
		}

		pkgLabels["type"] = labelTypeValue

		metric := registerer.NewGauge(prometheus.GaugeOpts{
			Name:        fmt.Sprintf("total_%s", metricName),
			Help:        "Total value for something",
			ConstLabels: pkgLabels,
		})

		m[fmt.Sprintf("total_%s", val)] = metric

		listOfMetrics = append(listOfMetrics, metricMapping{
			metric:     metric,
			index:      val,
			metricType: TurbostatTypes.Total,
		})
	}

	return listOfMetrics
}

func prependStringWithZeros(input string, length int) string {
	var newString string = ""

	requiredLength := int(math.Log10(float64(length))) + 1

	for i := 0; i < (requiredLength - len(input)); i++ {
		newString += "0"
	}

	return newString + input
}

func executeProgram(collectTimeSeconds int) bytes.Reader {
	var cmd *exec.Cmd

	if isCommandCat {
		cmd = exec.Command("cat", "sample2.csv")
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
	}

	//cmd.Wait()

	lines := bytes.Split(out.Bytes(), []byte("\n"))

	log.Trace().Msgf("Command output: %s", lines)

	if len(lines) < 2 {
		log.Info().Msg("No data to parse")
	}

	if !strings.HasPrefix(string(lines[0]), "Core") {
		lines = lines[1:]
	}

	reader := bytes.NewReader(bytes.Join(lines, []byte("\n")))

	return *reader
}

func parseOutput(input io.Reader) ([]string, []map[string]interface{}) {
	csvReader := csv.NewReader(input)
	csvReader.FieldsPerRecord = -1
	csvReader.Comma = '\t'

	// Core	CPU	Avg_MHz	Busy%	Bzy_MHz	TSC_MHz	IPC	IRQ	SMI	POLL (c states) POLL% (c% states) CPU%c1 CPU%c6 CPU%c7 CoreTmp CoreThr
	headers, err := csvReader.Read()
	if err != nil {
		log.Fatal().Err(err).Msg("")
	}

	var data []map[string]interface{}

	for {
		row, err := csvReader.Read()
		if err != nil {
			break
		}

		m := make(map[string]interface{})
		for i, val := range row {
			f, err := strconv.ParseFloat(val, 64)
			if err == nil {
				m[headers[i]] = f
				continue
			}

			b, err := strconv.ParseBool(val)
			if err == nil {
				m[headers[i]] = b
				continue
			}

			m[headers[i]] = val
		}

		data = append(data, m)
	}

	return headers, data
}

func Update() {
	reader := executeProgram(defaultSleepTimer)
	_, data := parseOutput(&reader)

	for _, val := range listOfMetrics {

		if tr, ok := val.metric.(prometheus.Gauge); ok {

			switch metricType := val.metricType; metricType {
			case TurbostatTypes.Total:
				vate, _ := strconv.ParseFloat(fmt.Sprintf("%.2f", data[0][val.index]), 2)
				tr.Set(vate)
			case TurbostatTypes.Core:
				vate, _ := strconv.ParseFloat(fmt.Sprintf("%.2f", data[val.coreIndex][val.index]), 2)
				tr.Set(vate)
			case TurbostatTypes.Cpu:
				vate, _ := strconv.ParseFloat(fmt.Sprintf("%.2f", data[val.coreIndex][val.index]), 2)
				tr.Set(vate)
			default:
				log.Info().Msgf("Unsupported metric %s", metricType)
			}
		} else {
			fmt.Println("gauge parsing failed")
		}

	}
}

type helloWorldhandler struct{}

func (h helloWorldhandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !isBackgroundMode {
		Update()
	}
	promhttp.Handler().ServeHTTP(w, r)
}

func basicAuth(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract the username and password from the request
		// Authorization header. If no Authentication header is present
		// or the header value is invalid, then the 'ok' return value
		// will be false.
		username, password, ok := r.BasicAuth()
		if ok {
			// Calculate SHA-256 hashes for the provided and expected
			// usernames and passwords.
			usernameHash := sha256.Sum256([]byte(username))
			passwordHash := sha256.Sum256([]byte(password))
			expectedUsernameHash := sha256.Sum256([]byte(basicAuthUsername))
			expectedPasswordHash := sha256.Sum256([]byte(basicAuthPassword))

			// Use the subtle.ConstantTimeCompare() function to check if
			// the provided username and password hashes equal the
			// expected username and password hashes. ConstantTimeCompare
			// will return 1 if the values are equal, or 0 otherwise.
			// Importantly, we should to do the work to evaluate both the
			// username and password before checking the return values to
			// avoid leaking information.
			usernameMatch := (subtle.ConstantTimeCompare(usernameHash[:], expectedUsernameHash[:]) == 1)
			passwordMatch := (subtle.ConstantTimeCompare(passwordHash[:], expectedPasswordHash[:]) == 1)

			// If the username and password are correct, then call
			// the next handler in the chain. Make sure to return
			// afterwards, so that none of the code below is run.
			if usernameMatch && passwordMatch {
				next.ServeHTTP(w, r)
				return
			}
		}

		// If the Authentication header is not present, is invalid, or the
		// username or password is wrong, then set a WWW-Authenticate
		// header to inform the client that we expect them to use basic
		// authentication and send a 401 Unauthorized response.
		w.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	})
}

var Version = "development"
var listOfMetrics []metricMapping = nil
var defaultSleepTimer = 5
var isCommandCat = false
var isBackgroundMode = false
var backgroundCollectSeconds = 30
var basicAuthUsername string
var basicAuthPassword string
var basicAuthEnabled = false
var listenAddr = "0.0.0.0:9101"

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
			defaultSleepTimer = convertVal
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

func main() {
	flagVersion := flag.Bool("version", false, "prints the version")

	flag.Parse()

	if *flagVersion {
		fmt.Println(Version)
		os.Exit(0)
	}

	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	startServer()
}

func startServer() {
	fmt.Println("Prometheus turbostat exporter - created by BlackDark (https://github.com/BlackDark/prometheus_turbostat_exporter)")
	parseConfiguration()

	reader := executeProgram(0)

	listOfMetrics = buildMetricList(&reader)

	if isBackgroundMode {
		log.Debug().Msgf("Starting ticker")
		ticker := time.NewTicker(time.Duration(backgroundCollectSeconds) * time.Second)
		quit := make(chan struct{})
		manualTick := make(chan bool)

		go func() {
			for {
				select {
				case <-ticker.C:
					log.Debug().Msgf("Ticker update")
					Update()
				case <-manualTick:
					log.Debug().Msgf("Manual tick update")
					Update()
				case <-quit:
					log.Debug().Msgf("Stop background updater")
					ticker.Stop()
					return
				}
			}
		}()
		manualTick <- true
	}

	var metricsHandler http.HandlerFunc = helloWorldhandler{}.ServeHTTP

	if basicAuthEnabled {
		metricsHandler = basicAuth(metricsHandler)
	}

	http.Handle("/metrics", metricsHandler)
	//http.Handle("/metrics", promhttp.Handler())
	log.Info().Msgf("Starting server on %s", listenAddr)
	log.Fatal().Err(http.ListenAndServe(listenAddr, nil)).Msg("")
}

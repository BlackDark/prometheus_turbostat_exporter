package main

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"maps"
	"math"
	"os"
	"os/exec"
	"slices"
	"strconv"
	"strings"
	"time"

	"net/http"

	"github.com/joho/godotenv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	//log "github.com/sirupsen/logrus"
	"go.uber.org/zap"
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

	logger.Infof("Extracted %d headers which will be extracted\n", headersLen)
	logger.Infof("Following headers will be used: %s \n", headers)
	logger.Infof("Extracted %d data lines\n", dataLen)

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

	logger.Infof("Total cores %d, Total cpus (threads) %d\n", len(coreIndices), len(cpuIndices))

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

	logger.Debugf("Extracted following header indices for states: Total %d, Core %d, CPU %d, Pkg %d", headerTotalIndices, headerCoreIndices, headerCpuIndices, headerPkgIndices)

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
	logger.Debugf("Executing command: %s", cmd.Args)

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	err := cmd.Run()
	if err != nil {
		logger.Fatalf("Failed to run turbostat: %v", err)
	}

	//cmd.Wait()

	lines := bytes.Split(out.Bytes(), []byte("\n"))

	logger.Debugf("Command output: %s", lines)

	if len(lines) < 2 {
		logger.Infoln("No data to parse")
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
		logger.Fatal(err)
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
				logger.Infof("Unsupported metric %s", metricType)
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

var listOfMetrics []metricMapping = nil
var defaultSleepTimer = 5
var isCommandCat = false
var isBackgroundMode = false
var backgroundCollectSeconds = 30
var logger *zap.SugaredLogger

func parseConfiguration() {
	godotenv.Load()

	newConf := zap.NewProductionConfig()
	if val, ok := os.LookupEnv("TURBOSTAT_EXPORTER_LOG_LEVEL"); ok {
		level, err := zap.ParseAtomicLevel(val)
		if err != nil {
			fmt.Printf("Invalid log level provided %s. Setting default to INFO", val)
		} else {
			newConf.Level = level
		}
	}
	zapLogger, _ := newConf.Build()
	defer zapLogger.Sync() // flushes buffer, if any
	logger = zapLogger.Sugar()

	// use the default if not set
	if val, ok := os.LookupEnv("TURBOSTAT_EXPORTER_DEFAULT_COLLECT_SECONDS"); ok {
		if convertVal, err := strconv.Atoi(val); err == nil {
			defaultSleepTimer = convertVal
		}
	}

	logger.Infof("Configured turbostat collecting time of %d seconds", defaultSleepTimer)

	if val, ok := os.LookupEnv("TURBOSTAT_EXPORTER_DEBUG_CAT_EXEC"); ok {
		if val == "true" {
			isCommandCat = true
			logger.Infof("Running in testing 'cat' mode. Will not execute turbostat.")
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
		logger.Infof("Running collector in background with interval %d.", backgroundCollectSeconds)
	} else {
		logger.Infof("Running collector in active mode (on request will execute turbostat)")
	}
}

func main() {
	fmt.Println("Prometheus turbostat exporter - created by BlackDark")
	parseConfiguration()

	reader := executeProgram(0)

	listOfMetrics = buildMetricList(&reader)

	if isBackgroundMode {
		logger.Debugf("Starting ticker")
		ticker := time.NewTicker(time.Duration(backgroundCollectSeconds) * time.Second)
		quit := make(chan struct{})
		manualTick := make(chan bool)

		go func() {
			for {
				select {
				case <-ticker.C:
					logger.Debugf("Ticker update")
					Update()
				case <-manualTick:
					logger.Debugf("Manual tick update")
					Update()
				case <-quit:
					logger.Debugf("Stop background updater")
					ticker.Stop()
					return
				}
			}
		}()
		manualTick <- true
	}

	http.Handle("/metrics", helloWorldhandler{})
	//http.Handle("/metrics", promhttp.Handler())
	logger.Fatal(http.ListenAndServe(":9101", nil))
}

package main

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"maps"
	"os"
	"os/exec"
	"slices"
	"strconv"
	"strings"

	"bufio"

	"github.com/gocarina/gocsv"

	"net/http"

	"github.com/joho/godotenv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
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

	log.Printf("Extracted %d headers which will be extracted\n", headersLen)
	log.Printf("Following headers will be used: %s \n", headers)
	log.Printf("Extracted %d data lines\n", dataLen)

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

	log.Printf("Total cores %d, Total cpus (threads) %d\n", len(coreIndices), len(cpuIndices))

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
		if (i < pollIndex) || (i >= coreTmpIndex && i < firstPkgState) || (i >= lastPkgState && i < headersLen) {
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

	log.Debugf("Extracted following header indices for states: Total %d, Core %d, CPU %d, Pkg %d", headerTotalIndices, headerCoreIndices, headerCpuIndices, headerPkgIndices)

	for _, i := range headerTotalIndices {
		val := headers[i]

		convertedVal := strings.ReplaceAll(val, "%", "_percent")
		convertedVal = strings.ToLower(convertedVal)

		pkgLabels := make(prometheus.Labels)

		pkgLabels["type"] = convertedVal

		metric := promauto.NewGauge(prometheus.GaugeOpts{
			Name:        fmt.Sprintf("total_info"),
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

		convertedVal := strings.ReplaceAll(val, "%", "_percent")
		convertedVal = strings.ToLower(convertedVal)

		pkgLabels := make(prometheus.Labels)

		pkgLabels["type"] = convertedVal

		metric := promauto.NewGauge(prometheus.GaugeOpts{
			Name:        fmt.Sprintf("total_core_states"),
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
				"num_core": strconv.Itoa(int(data[coreV]["CPU"].(float64))),
			}
			maps.Copy(coreLabels, pkgLabels)

			listOfMetrics = append(listOfMetrics, metricMapping{
				metric: promauto.NewGauge(prometheus.GaugeOpts{
					Name:        fmt.Sprintf("core_states"),
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

		convertedVal := strings.ReplaceAll(val, "%", "_percent")
		convertedVal = strings.ToLower(convertedVal)

		pkgLabels := make(prometheus.Labels)

		pkgLabels["type"] = convertedVal

		metric := promauto.NewGauge(prometheus.GaugeOpts{
			Name:        fmt.Sprintf("total_cpu_states_percent"),
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
				"num_cpu": strconv.Itoa(int(data[cpuV]["CPU"].(float64))),
			}
			maps.Copy(coreLabels, pkgLabels)

			listOfMetrics = append(listOfMetrics, metricMapping{
				metric: promauto.NewGauge(prometheus.GaugeOpts{
					Name:        fmt.Sprintf("cpu_state"),
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

		convertedVal := strings.ReplaceAll(val, "%", "_percent")
		convertedVal = strings.ToLower(convertedVal)

		pkgLabels := make(prometheus.Labels)

		pkgLabels["type"] = convertedVal

		metric := promauto.NewGauge(prometheus.GaugeOpts{
			Name:        fmt.Sprintf("total_pkg_states_percent"),
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

func executeProgram(collectTimeSeconds int) bytes.Reader {
	var cmd *exec.Cmd

	if isCommandCat {
		cmd = exec.Command("cat", "prox.csv")
	} else {
		cmd = exec.Command("turbostat", "--quiet", "sleep", strconv.Itoa(collectTimeSeconds))
	}
	log.Debugf("Executing command: %s", cmd.Args)

	var out bytes.Buffer
	cmd.Stdout = &out

	err := cmd.Run()
	if err != nil {
		log.Fatalf("Failed to run turbostat: %v", err)
	}

	cmd.Wait()

	lines := bytes.Split(out.Bytes(), []byte("\n"))

	log.Debugf("Command output: %s", lines)

	if len(lines) < 2 {
		log.Println("No data to parse")
	}

	reader := bytes.NewReader(out.Bytes())

	return *reader
}

func parseOutput(input io.Reader) ([]string, []map[string]interface{}) {
	csvReader := csv.NewReader(input)
	csvReader.FieldsPerRecord = -1
	csvReader.Comma = '\t'

	// Core	CPU	Avg_MHz	Busy%	Bzy_MHz	TSC_MHz	IPC	IRQ	SMI	POLL (c states) POLL% (c% states) CPU%c1 CPU%c6 CPU%c7 CoreTmp CoreThr
	headers, err := csvReader.Read()
	if err != nil {
		log.Fatal(err)
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
	// TODO maybe run the execution in the background and just return the current values?
	reader := executeProgram(defaultSleepTimer)
	_, data := parseOutput(&reader)

	for _, val := range listOfMetrics {

		if tr, ok := val.metric.(prometheus.Gauge); ok {

			switch metricType := val.metricType; metricType {
			case TurbostatTypes.Total:
				vate, _ := strconv.ParseFloat(fmt.Sprintf("%.2f", data[0][val.index]), 2)
				tr.Set(vate)
				//tr.SetToCurrentTime()
				// log.Printf("Setting value for %s = %f", val.index, vate)
			case TurbostatTypes.Core:
				vate, _ := strconv.ParseFloat(fmt.Sprintf("%.2f", data[val.coreIndex][val.index]), 2)
				tr.Set(vate)
				//tr.SetToCurrentTime()
				// log.Printf("Setting value for %s = %f", val.index, vate)
			case TurbostatTypes.Cpu:
				vate, _ := strconv.ParseFloat(fmt.Sprintf("%.2f", data[val.coreIndex][val.index]), 2)
				tr.Set(vate)
				//tr.SetToCurrentTime()
				// log.Printf("Setting value for %s = %f", val.index, vate)
			default:
				log.Printf("Unsupported metric %s", metricType)
			}
		} else {
			fmt.Println("s is not a string")
		}

	}
}

type helloWorldhandler struct{}

func (h helloWorldhandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	Update()
	promhttp.Handler().ServeHTTP(w, r)
}

var listOfMetrics []metricMapping = nil
var defaultSleepTimer int = 5
var isCommandCat = false

func main() {
	godotenv.Load()

	if val, ok := os.LookupEnv("TURBOSTAT_EXPORTER_LOG_LEVEL"); ok {
		switch val {
		case "debug":
			log.SetLevel(log.DebugLevel)
		default:
			log.SetLevel(log.InfoLevel)
		}
	}

	fmt.Println("Prometheus turbostat exporter - created by BlackDark")

	// use the default if not set
	if val, ok := os.LookupEnv("TURBOSTAT_EXPORTER_DEFAULT_COLLECT_SECONDS"); ok {
		if convertVal, err := strconv.Atoi(val); err == nil {
			defaultSleepTimer = convertVal
		}
	}

	if val, ok := os.LookupEnv("TURBOSTAT_EXPORTER_DEBUG_CAT_EXEC"); ok {
		if val == "true" {
			isCommandCat = true
		}
	}

	log.Printf("Configured turbostat collecting time of %d seconds", defaultSleepTimer)

	// TODO could be optimized
	reader := executeProgram(1)

	// buf := make([]byte, 4)
	// n, err := reader.Read(buf)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// log.Println(string(buf[:n]))

	// reader.Seek(0, 0)
	// foo := newFooCollector2(&reader)
	//prometheus.MustRegister(foo)

	listOfMetrics = buildMetricList(&reader)

	//foo := newFooCollector()

	http.Handle("/console/metrics", helloWorldhandler{})
	//http.Handle("/console/metrics", promhttp.Handler())
	log.Fatal(http.ListenAndServe(":9101", nil))

}

func exec3() {

	header, data := test3()
	fmt.Printf("%v\n", header)
	fmt.Printf("%s\n", header[0])

	for i, j := range data {
		fmt.Println(i, j)

		fmt.Printf("DIGGA_%f{label=pkg}\n", j["PkgWatt"])
	}
}

func test1() {
	clientsFile, err := os.OpenFile("test.csv", os.O_RDWR|os.O_CREATE, os.ModePerm)
	if err != nil {
		panic(err)
	}
	defer clientsFile.Close()

	// Set up the CSV reader with tab as the delimiter
	gocsv.SetCSVReader(func(in io.Reader) gocsv.CSVReader {
		r := csv.NewReader(in)
		r.Comma = '\t'
		return r
	})

	// Use gocsv to read the rest of the file into maps
	maps, err := gocsv.CSVToMaps(clientsFile)
	if err != nil {
		panic(err)
	}

	if err != nil { // Load clients from file
		panic(err)
	}

	for _, client := range maps {
		fmt.Println("Hello", client)
	}
}

func test2() []string {
	in, err := os.Open("test.csv")
	if err != nil {
		log.Fatalf("%s", err)
	}
	defer in.Close()

	// Set up the CSV reader with tab as the delimiter
	gocsv.SetCSVReader(func(in io.Reader) gocsv.CSVReader {
		r := csv.NewReader(in)
		r.Comma = '\t'
		return r
	})

	m, err := gocsv.CSVToMaps(bufio.NewReader(in))
	if err != nil {
		log.Fatalf("%s", err)
	}

	header := []string{}
	for k := range m[0] {
		header = append(header, k)
	}

	return header
}

func test3() ([]string, []map[string]interface{}) {
	in, err := os.Open("prox.csv")
	if err != nil {
		log.Fatalf("%s", err)
	}
	defer in.Close()

	csvReader := csv.NewReader(in)
	csvReader.FieldsPerRecord = -1
	csvReader.Comma = '\t'

	// Core	CPU	Avg_MHz	Busy%	Bzy_MHz	TSC_MHz	IPC	IRQ	SMI	POLL (c states) POLL% (c% states) CPU%c1 CPU%c6 CPU%c7 CoreTmp CoreThr
	headers, err := csvReader.Read()
	if err != nil {
		log.Fatal(err)
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

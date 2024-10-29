package main

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"math/rand/v2"
	"os"
	"os/exec"
	"slices"
	"strconv"
	"strings"
	"time"

	"bufio"

	"github.com/gocarina/gocsv"

	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Define a struct for you collector that contains pointers
// to prometheus descriptors for each metric you wish to expose.
// Note you can also include fields of other types if they provide utility
// but we just won't be exposing them as metrics.
type fooCollector struct {
	fooMetric  *prometheus.Desc
	barMetric  *prometheus.Desc
	allMetrics map[string]prometheus.Metric
	testMetric []metricMapping
}

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
	metricType TurbostatType
}

// You must create a constructor for you collector that
// initializes every descriptor and returns a pointer to the collector
func newFooCollector() *fooCollector {
	return &fooCollector{
		fooMetric: prometheus.NewDesc("foo_metric",
			"Shows whether a foo has occurred in our cluster",
			nil, nil,
		),
		barMetric: prometheus.NewDesc("bar_metric",
			"Shows whether a bar has occurred in our cluster",
			nil, nil,
		),
	}
}

// You must create a constructor for you collector that
// initializes every descriptor and returns a pointer to the collector
func newFooCollector2(reader io.Reader) *fooCollector {
	headers, _ := parseOutput(reader)
	headersLen := len(headers)

	log.Printf("Extracted %d headers which will be extracted\n", headersLen)
	log.Printf("Following headers will be used: %s \n", headers)

	pollIndex := slices.Index(headers, "POLL")
	pollPIndex := slices.Index(headers, "POLL%")
	firstCpuPercent := slices.IndexFunc(headers, func(n string) bool {
		return strings.HasPrefix(n, "CPU%")
	})
	coreTmpIndex := slices.Index(headers, "CoreTmp")
	firstPkgState := slices.Index(headers, "CPUGFX%") + 1
	lastPkgState := slices.Index(headers, "CPU%LPI") - 1

	m := make(map[string]prometheus.Metric)
	listOfMetrics := []metricMapping{}

	// total_info
	for i := 2; i < pollIndex; i++ {
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

	// total_info
	for i := coreTmpIndex; i < firstPkgState; i++ {
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

	// total_info
	for i := lastPkgState + 1; i < headersLen; i++ {
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

	// total_core_states
	for i := pollIndex; i < pollPIndex; i++ {
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
	}

	// total_core_states_percent
	for i := pollPIndex; i < firstCpuPercent; i++ {
		val := headers[i]

		convertedVal := strings.ReplaceAll(val, "%", "_percent")
		convertedVal = strings.ToLower(convertedVal)

		pkgLabels := make(prometheus.Labels)

		pkgLabels["type"] = convertedVal

		metric := promauto.NewGauge(prometheus.GaugeOpts{
			Name:        fmt.Sprintf("total_core_states_percent"),
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

	// total_cpu_states_percent
	for i := firstCpuPercent; i < coreTmpIndex; i++ {
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

	}

	// total_pkg_states_percent
	for i := firstPkgState; i <= lastPkgState; i++ {
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

	// total
	// for i, val := range headers {
	// 	if i < 2 {
	// 		continue
	// 	}

	// 	convertedVal := strings.ReplaceAll(val, "%", "_percent")
	// 	convertedVal = strings.ToLower(convertedVal)

	// 	pkgLabels := make(prometheus.Labels)

	// 	pkgLabels["type"] = convertedVal

	// 	m[fmt.Sprintf("total_%s", val)] = promauto.NewGauge(prometheus.GaugeOpts{
	// 		Name:        fmt.Sprintf("total"),
	// 		Help:        "Total value for something",
	// 		ConstLabels: pkgLabels,
	// 	})

	// 	fmt.Println(i, val, convertedVal)
	// }

	// other
	// for i, val := range headers {
	// 	if i < 2 {
	// 		continue
	// 	}

	// 	convertedVal := strings.ReplaceAll(val, "%", "percent")
	// 	convertedVal = strings.ToLower(convertedVal)

	// 	pkgLabels := make(prometheus.Labels)

	// 	pkgLabels["type"] = "pkg"

	// 	m[fmt.Sprintf("total_%s", val)] = promauto.NewGauge(prometheus.GaugeOpts{
	// 		Name:        fmt.Sprintf("total_%s", convertedVal),
	// 		Help:        "Total value for something",
	// 		ConstLabels: pkgLabels,
	// 	})

	// 	fmt.Println(i, val, convertedVal)
	// }

	return &fooCollector{
		allMetrics: m,
		testMetric: listOfMetrics,
		fooMetric: prometheus.NewDesc("foo_metric",
			"Shows whether a foo has occurred in our cluster",
			nil, nil,
		),
		barMetric: prometheus.NewDesc("bar_metric",
			"Shows whether a bar has occurred in our cluster",
			nil, nil,
		),
	}
}

func buildMetricList(reader io.Reader) []metricMapping {
	headers, _ := parseOutput(reader)
	headersLen := len(headers)

	log.Printf("Extracted %d headers which will be extracted\n", headersLen)
	log.Printf("Following headers will be used: %s \n", headers)

	pollIndex := slices.Index(headers, "POLL")
	pollPIndex := slices.Index(headers, "POLL%")
	firstCpuPercent := slices.IndexFunc(headers, func(n string) bool {
		return strings.HasPrefix(n, "CPU%")
	})
	coreTmpIndex := slices.Index(headers, "CoreTmp")
	firstPkgState := slices.Index(headers, "CPUGFX%") + 1
	lastPkgState := slices.Index(headers, "CPU%LPI") - 1

	m := make(map[string]prometheus.Metric)
	listOfMetrics := []metricMapping{}

	// total_info
	for i := 2; i < pollIndex; i++ {
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

	// total_info
	for i := coreTmpIndex; i < firstPkgState; i++ {
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

	// total_info
	for i := lastPkgState + 1; i < headersLen; i++ {
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

	// total_core_states
	for i := pollIndex; i < pollPIndex; i++ {
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
	}

	// total_core_states_percent
	for i := pollPIndex; i < firstCpuPercent; i++ {
		val := headers[i]

		convertedVal := strings.ReplaceAll(val, "%", "_percent")
		convertedVal = strings.ToLower(convertedVal)

		pkgLabels := make(prometheus.Labels)

		pkgLabels["type"] = convertedVal

		metric := promauto.NewGauge(prometheus.GaugeOpts{
			Name:        fmt.Sprintf("total_core_states_percent"),
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

	// total_cpu_states_percent
	for i := firstCpuPercent; i < coreTmpIndex; i++ {
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

	}

	// total_pkg_states_percent
	for i := firstPkgState; i <= lastPkgState; i++ {
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
	cmd := exec.Command("cat", "prox.csv")
	time.Sleep(time.Duration(collectTimeSeconds) * time.Second)
	var out bytes.Buffer
	cmd.Stdout = &out

	err := cmd.Run()
	if err != nil {
		log.Fatalf("Failed to run turbostat: %v", err)
	}

	lines := bytes.Split(out.Bytes(), []byte("\n"))

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

// Each and every collector must implement the Describe function.
// It essentially writes all descriptors to the prometheus desc channel.
func (collector *fooCollector) Describe(ch chan<- *prometheus.Desc) {

	//Update this section with the each metric you create for a given collector
	ch <- collector.fooMetric
	ch <- collector.barMetric
}

// Collect implements required collect function for all promehteus collectors
func (collector *fooCollector) Collect(ch chan<- prometheus.Metric) {

	// TODO maybe run the execution in the background and just return the current values?
	reader := executeProgram(1)
	_, data := parseOutput(&reader)

	for _, val := range collector.testMetric {

		if tr, ok := val.metric.(prometheus.Gauge); ok {

			switch metricType := val.metricType; metricType {
			case TurbostatTypes.Total:
				vate, _ := strconv.ParseFloat(fmt.Sprintf("%.2f", data[0][val.index]), 32)
				tr.Set(vate)
				tr.SetToCurrentTime()
				log.Printf("Setting value for %s = %f", val.index, vate)
			default:
				log.Printf("Unsupported metric %s", metricType)
			}
		} else {
			fmt.Println("s is not a string")
		}

	}

	//Implement logic here to determine proper metric value to return to prometheus
	//for each descriptor or call other functions that do so.
	var metricValue float64
	if 1 == 1 {
		metricValue += rand.Float64()
	}
	//Write latest value for each metric in the prometheus metric channel.
	//Note that you can pass CounterValue, GaugeValue, or UntypedValue types here.
	m1 := prometheus.MustNewConstMetric(collector.fooMetric, prometheus.GaugeValue, metricValue)
	m2 := prometheus.MustNewConstMetric(collector.barMetric, prometheus.GaugeValue, metricValue)
	m1 = prometheus.NewMetricWithTimestamp(time.Now().Add(-time.Hour), m1)
	m2 = prometheus.NewMetricWithTimestamp(time.Now(), m2)

	ch <- m1
	ch <- m2

	// vate, _ := strconv.ParseFloat(fmt.Sprintf("%.2f", metricValue), 32)

	// for _, val := range collector.allMetrics {
	// 	if tr, ok := val.(prometheus.Gauge); ok {
	// 		tr.Set(vate)
	// 		tr.SetToCurrentTime()
	// 		//ch <- tr
	// 	} else {
	// 		fmt.Println("s is not a string")
	// 	}

	// }
}

func Update() {
	// TODO maybe run the execution in the background and just return the current values?
	reader := executeProgram(1)
	_, data := parseOutput(&reader)

	for _, val := range listOfMetrics {

		if tr, ok := val.metric.(prometheus.Gauge); ok {

			switch metricType := val.metricType; metricType {
			case TurbostatTypes.Total:
				vate, _ := strconv.ParseFloat(fmt.Sprintf("%.2f", data[0][val.index]), 2)
				tr.Set(vate)
				//tr.SetToCurrentTime()
				log.Printf("Setting value for %s = %f", val.index, vate)
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

func main() {
	fmt.Println("Prometheus turbostat exporter - created by BlackDark")
	log.Printf("Configured turbostat collecting time of %d seconds", 3)

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

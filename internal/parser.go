package internal

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
)

type TurbostatParser struct {
	// probably to complex for such a simple thing
	colParsers []columnParseFunc
	categories map[int]string
}

type columnParseFunc func(row *TurbostatRow, col string)

func NewTurbostatParser() *TurbostatParser {
	parser := &TurbostatParser{}
	return parser
}

func (p *TurbostatParser) SetupColumnParsers(headers []string) {
	if len(p.colParsers) > 0 {
		// parsers are already setup
		return
	}

	col := 0
	parsers := []columnParseFunc{}

	// Package column does only appear on systems with multiple CPU sockets
	if headers[col] == "Package" {
		parsers = append(parsers, func(r *TurbostatRow, c string) { r.Pkg = c })
		col++
	}

	if headers[col] == "Core" {
		parsers = append(parsers, func(r *TurbostatRow, c string) { r.Core = c })
		col++
	}

	if headers[col] == "CPU" {
		parsers = append(parsers, func(r *TurbostatRow, c string) { r.Cpu = c })
		col++
	}

	for ; col < len(headers) && headers[col] != "POLL"; col++ {
		parsers = append(parsers, p.createDefaultColumnParser(headers[col]))
	}

	for ; col < len(headers) && headers[col] != "POLL%"; col++ {
		parsers = append(parsers, p.createColumnParser(func(r *TurbostatRow) map[string]float64 { return r.CpuStates }, headers[col]))
	}

	for ; col < len(headers) && !strings.HasPrefix(headers[col], "CPU"); col++ {
		parsers = append(parsers, p.createColumnParser(func(r *TurbostatRow) map[string]float64 { return r.CpuStatesPercent }, headers[col]))
	}

	for ; col < len(headers) && !strings.HasPrefix(headers[col], "Core"); col++ {
		parsers = append(parsers, p.createColumnParser(func(r *TurbostatRow) map[string]float64 { return r.CoreStatesPercent }, headers[col]))
	}

	for ; col < len(headers) && !strings.Contains(headers[col], "%pc"); col++ {
		parsers = append(parsers, p.createDefaultColumnParser(headers[col]))
	}

	for ; col < len(headers) && !strings.HasPrefix(headers[col], "CPU"); col++ {
		parsers = append(parsers, p.createColumnParser(func(r *TurbostatRow) map[string]float64 { return r.PkgStatesPercent }, headers[col]))
	}

	for ; col < len(headers); col++ {
		parsers = append(parsers, p.createDefaultColumnParser(headers[col]))
	}

	p.colParsers = parsers
}

func (p *TurbostatParser) createColumnParser(mapper func(r *TurbostatRow) map[string]float64, header string) columnParseFunc {
	return func(row *TurbostatRow, c string) {
		key := sanitizeHeader(header)
		val, err := strconv.ParseFloat(c, 64)
		if err != nil {
			log.Warn().Msgf("Failed to parse column %s with value %q", header, c)
			return
		}

		m := mapper(row)
		if _, ok := m[key]; ok {
			log.Error().Msgf("Found duplicate key in column %q: %q", key, header)
		}
		m[key] = val
	}
}

func (p *TurbostatParser) createDefaultColumnParser(header string) columnParseFunc {
	if strings.Contains(header, "%") {
		return p.createColumnParser(func(r *TurbostatRow) map[string]float64 { return r.OtherPercent }, header)
	} else {
		return p.createColumnParser(func(r *TurbostatRow) map[string]float64 { return r.Other }, header)
	}
}

func (p *TurbostatParser) ParseRow(rowData []string) (*TurbostatRow, error) {
	if len(rowData) > len(p.colParsers) {
		return nil, fmt.Errorf("found %d columns but have only %d parsers", len(rowData), len(p.colParsers))
	}

	row := NewTurbostatRow()
	for i, value := range rowData {
		parser := p.colParsers[i]
		parser(row, value)
	}
	return row, nil
}

func (p *TurbostatParser) ParseRows(inputRows [][]string) ([]TurbostatRow, error) {
	rows := make([]TurbostatRow, 0, len(inputRows))
	for _, inputRow := range inputRows {
		row, err := p.ParseRow(inputRow)
		if err != nil {
			return nil, err
		}
		rows = append(rows, *row)
	}
	return rows, nil
}

func sanitizeHeader(h string) string {
	res := strings.ReplaceAll(h, "%", "")
	res = strings.ToLower(res)
	return res
}

// help struct to identify what information a row contains
func (p *TurbostatParser) ParseCategories(headers []string, rows [][]string) map[int]string {
	if p.categories != nil {
		return p.categories
	}

	lengthMap := make(map[int]int)
	categoryMap := map[int]string{}

	if len(headers) == 0 || len(rows) == 0 {
		return categoryMap
	}

	for _, row := range rows {
		if len(row) > 0 && row[0] == "-" {
			// Total row
			continue
		}
		lengthMap[len(row)]++
	}

	var lengths []int

	for l := range lengthMap {
		lengths = append(lengths, l)
	}
	sort.Ints(lengths)

	switch len(lengths) {
	case 4:
		categoryMap[lengths[3]] = "total"
		categoryMap[lengths[2]] = "package"
		categoryMap[lengths[1]] = "core"
		categoryMap[lengths[0]] = "cpu"
	case 3:
		categoryMap[lengths[2]] = "package"
		categoryMap[lengths[1]] = "core"
		categoryMap[lengths[0]] = "cpu"
	case 2:
		categoryMap[lengths[1]] = "core"
		categoryMap[lengths[0]] = "cpu"
	default:
		categoryMap[lengths[len(lengths)-1]] = "total"
		categoryMap[lengths[0]] = "cpu"
	}

	p.categories = categoryMap

	return categoryMap
}

func (p *TurbostatParser) ParseRowSimple(category string, headers []string, row []string) map[string][]*TurbostatRow {
	result := map[string][]*TurbostatRow{
		"total":   {},
		"package": {},
		"core":    {},
		"cpu":     {},
	}

	tr := NewTurbostatRow()
	tr.Category = category

	// Use headers for key mapping
	col := 0

	if category != "total" {
		if headers[0] == "Package" {
			tr.Pkg = row[0]
			tr.Core = row[1]
			tr.Cpu = row[2]
			col = 3
		} else {
			tr.Core = row[0]
			tr.Cpu = row[1]
			col = 2
		}
	}

	for i := col; i < len(row); i++ {
		val, err := strconv.ParseFloat(row[i], 64)
		if err != nil {
			continue
		}
		if i < len(headers) {
			key := headers[i]
			if strings.Contains(key, "%") {
				tr.OtherPercent[key] = val
			} else {
				tr.Other[key] = val
			}
		}
	}
	result[category] = append(result[category], tr)

	// add duplicates where necessary
	switch category {
	case "package":
		coreResult := *tr
		coreResult.Category = "core"
		result["core"] = append(result["core"], &coreResult)

		cpuResult := *tr
		cpuResult.Category = "cpu"
		result["cpu"] = append(result["cpu"], &cpuResult)
	case "core":
		cpuResult := *tr
		cpuResult.Category = "cpu"
		result["cpu"] = append(result["cpu"], &cpuResult)
	}

	return result
}

func (p *TurbostatParser) ParseRowsSimple(headers []string, rows [][]string) map[string][]*TurbostatRow {
	result := map[string][]*TurbostatRow{
		"total":   {},
		"package": {},
		"core":    {},
		"cpu":     {},
	}

	if len(headers) == 0 || len(rows) == 0 {
		return result
	}

	categoryMap := p.ParseCategories(headers, rows)

	for _, row := range rows {
		category := categoryMap[len(row)]

		if len(row) > 0 && row[0] == "-" {
			category = "total"
		}

		parsedRow := p.ParseRowSimple(category, headers, row)

		// merge maps
		for k, v := range parsedRow {
			result[k] = append(result[k], v...)
		}
	}
	return result
}

// old?
func ParseTurbostatOutput(raw string) ([]string, [][]string, error) {
	var headers []string
	var rows [][]string

	lines := strings.Split(raw, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		// If headers not set, use this line as headers
		if len(headers) == 0 {
			headers = fields
			continue
		}
		// Skip repeated headers in the data
		if strings.Join(fields, " ") == strings.Join(headers, " ") {
			continue
		}
		rows = append(rows, fields)
	}
	if len(headers) == 0 {
		return nil, nil, fmt.Errorf("no headers found in turbostat output")
	}
	return headers, rows, nil
}

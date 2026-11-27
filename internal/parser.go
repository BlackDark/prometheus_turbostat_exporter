package internal

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
)

type TurbostatParser struct {
	rowLengthToCategory map[int]string
	categoryToRowLength map[string]int
}

func NewTurbostatParser() *TurbostatParser {
	return &TurbostatParser{}
}

// topologyColumns are the leading identifier columns turbostat prints before
// any counter; which ones appear depends on the turbostat version and system.
var topologyColumns = map[string]bool{
	"Package": true,
	"Die":     true,
	"L3":      true,
	"Node":    true,
	"Module":  true,
	"Core":    true,
	"CPU":     true,
	"APIC":    true,
	"X2APIC":  true,
}

// isTurbostatHeaderLine checks whether fields look like a genuine turbostat
// header row, which always starts with a topology column.
func isTurbostatHeaderLine(fields []string) bool {
	return len(fields) > 0 && topologyColumns[fields[0]]
}

func countTopologyColumns(headers []string) int {
	n := 0
	for n < len(headers) && topologyColumns[headers[n]] {
		n++
	}
	return n
}

// normalizeTopologyID keeps label values stable across turbostat versions:
// turbostat 2026.04.21+ prints Core (and Module) IDs as "0x%x" instead of "%d".
func normalizeTopologyID(id string) string {
	hex, ok := strings.CutPrefix(id, "0x")
	if !ok {
		return id
	}
	v, err := strconv.ParseUint(hex, 16, 64)
	if err != nil {
		return id
	}
	return strconv.FormatUint(v, 10)
}

func sanitizeHeader(h string) string {
	res := strings.ReplaceAll(h, "%", "")
	res = strings.ToLower(res)
	return res
}

// help struct to identify what information a row contains
func (p *TurbostatParser) ParseCategories(headers []string, rows [][]string) map[int]string {
	if p.rowLengthToCategory != nil {
		return p.rowLengthToCategory
	}

	lengthMap := make(map[int]int)
	rowLengthToCategory := map[int]string{}
	categoryToRowLength := map[string]int{}

	if len(headers) == 0 || len(rows) == 0 {
		return rowLengthToCategory
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

	log.Debug().Msgf("Found %d different row lengths: %v", len(lengths), lengths)

	switch len(lengths) {
	case 4:
		rowLengthToCategory[lengths[3]] = "total"
		rowLengthToCategory[lengths[2]] = "package"
		rowLengthToCategory[lengths[1]] = "core"
		rowLengthToCategory[lengths[0]] = "cpu"

		categoryToRowLength["total"] = lengths[3]
		categoryToRowLength["package"] = lengths[2]
		categoryToRowLength["core"] = lengths[1]
		categoryToRowLength["cpu"] = lengths[0]
	case 3:
		rowLengthToCategory[lengths[2]] = "package"
		rowLengthToCategory[lengths[1]] = "core"
		rowLengthToCategory[lengths[0]] = "cpu"

		categoryToRowLength["total"] = lengths[2]
		categoryToRowLength["package"] = lengths[2]
		categoryToRowLength["core"] = lengths[1]
		categoryToRowLength["cpu"] = lengths[0]
	default:
		rowLengthToCategory[lengths[len(lengths)-1]] = "total"
		rowLengthToCategory[lengths[len(lengths)-1]] = "package"
		rowLengthToCategory[lengths[len(lengths)-1]] = "core"
		rowLengthToCategory[lengths[0]] = "cpu"

		categoryToRowLength["total"] = lengths[len(lengths)-1]
		categoryToRowLength["package"] = lengths[len(lengths)-1]
		categoryToRowLength["core"] = lengths[len(lengths)-1]
		categoryToRowLength["cpu"] = lengths[0]
	}

	for k, v := range categoryToRowLength {
		log.Debug().Msgf("Mapped category %s to length: %d", k, v)
	}

	p.rowLengthToCategory = rowLengthToCategory
	p.categoryToRowLength = categoryToRowLength

	return rowLengthToCategory
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

	col := countTopologyColumns(headers)

	if category != "total" {
		for i := 0; i < col && i < len(row); i++ {
			switch headers[i] {
			case "Package":
				tr.Pkg = normalizeTopologyID(row[i])
			case "Core":
				tr.Core = normalizeTopologyID(row[i])
			case "CPU":
				tr.CPU = normalizeTopologyID(row[i])
			}
		}
	}

	var (
		cpuResult     *TurbostatRow
		coreResult    *TurbostatRow
		packageResult *TurbostatRow
	)

	for i := col; i < len(row); i++ {
		val, err := strconv.ParseFloat(row[i], 64)
		if err != nil || math.IsNaN(val) {
			continue
		}
		// round to 2 decimal places
		val = math.Round(val*100) / 100

		if i > p.categoryToRowLength["cpu"] && cpuResult == nil {
			cpuResult = tr.CloneWithCategory("cpu")
		}
		if i > p.categoryToRowLength["core"] && coreResult == nil {
			coreResult = tr.CloneWithCategory("core")
		}
		if i > p.categoryToRowLength["package"] && packageResult == nil {
			packageResult = tr.CloneWithCategory("package")
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
		if coreResult != nil {
			result["core"] = append(result["core"], coreResult)
		}
		if cpuResult != nil {
			result["cpu"] = append(result["cpu"], cpuResult)
		}
	case "core":
		if cpuResult != nil {
			result["cpu"] = append(result["cpu"], cpuResult)
		}
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

func ParseTurbostatOutput(raw string) ([]string, [][]string, error) {
	var headers []string
	var rows [][]string

	lines := strings.SplitSeq(raw, "\n")
	for line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}

		if fields[len(fields)-1] == "sec" {
			// skip line contains execution time
			continue
		}

		// If headers not set, use this line as headers - but only if it actually
		// looks like a turbostat header. turbostat prints warnings (e.g. about
		// insufficient privileges to access /dev/cpu/*/msr) on the same stream
		// before the real header line, and those must not be mistaken for it.
		if len(headers) == 0 {
			if !isTurbostatHeaderLine(fields) {
				log.Warn().Msgf("Ignoring unexpected line before turbostat header: %s", line)
				continue
			}
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

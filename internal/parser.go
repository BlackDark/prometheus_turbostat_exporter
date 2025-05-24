package internal

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
)

type TurbostatParser struct {
	colParsers []columnParseFunc
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
	rows := []TurbostatRow{}
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

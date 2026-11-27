package internal

import "maps"

type TurbostatRow struct {
	Pkg          string
	Core         string
	CPU          string
	Other        map[string]float64
	OtherPercent map[string]float64
	Category     string // "total", "package", "core", "cpu"
}

func NewTurbostatRow() *TurbostatRow {
	return &TurbostatRow{
		Pkg:          "0",
		Other:        map[string]float64{},
		OtherPercent: map[string]float64{},
	}
}

func (r *TurbostatRow) CloneWithCategory(category string) *TurbostatRow {
	if r == nil {
		return nil
	}
	clone := &TurbostatRow{
		Pkg:          r.Pkg,
		Core:         r.Core,
		CPU:          r.CPU,
		Other:        make(map[string]float64, len(r.Other)),
		OtherPercent: make(map[string]float64, len(r.OtherPercent)),
		Category:     category,
	}
	maps.Copy(clone.Other, r.Other)
	maps.Copy(clone.OtherPercent, r.OtherPercent)
	return clone
}

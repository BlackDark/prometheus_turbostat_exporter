package internal

import "maps"

type TurbostatRow struct {
	Pkg               string
	Core              string
	Cpu               string
	CoreStatesPercent map[string]float64
	CpuStates         map[string]float64
	CpuStatesPercent  map[string]float64
	Other             map[string]float64
	OtherPercent      map[string]float64
	PkgStatesPercent  map[string]float64
	Category          string // "total", "package", "core", "cpu"
}

func NewTurbostatRow() *TurbostatRow {
	return &TurbostatRow{
		Pkg:               "0",
		CoreStatesPercent: map[string]float64{},
		CpuStates:         map[string]float64{},
		CpuStatesPercent:  map[string]float64{},
		Other:             map[string]float64{},
		OtherPercent:      map[string]float64{},
		PkgStatesPercent:  map[string]float64{},
	}
}

func (r *TurbostatRow) CloneWithCategory(category string) *TurbostatRow {
	if r == nil {
		return nil
	}
	clone := &TurbostatRow{
		Pkg:               r.Pkg,
		Core:              r.Core,
		Cpu:               r.Cpu,
		CoreStatesPercent: make(map[string]float64, len(r.CoreStatesPercent)),
		CpuStates:         make(map[string]float64, len(r.CpuStates)),
		CpuStatesPercent:  make(map[string]float64, len(r.CpuStatesPercent)),
		Other:             make(map[string]float64, len(r.Other)),
		OtherPercent:      make(map[string]float64, len(r.OtherPercent)),
		PkgStatesPercent:  make(map[string]float64, len(r.PkgStatesPercent)),
		Category:          category,
	}
	maps.Copy(clone.CoreStatesPercent, r.CoreStatesPercent)
	maps.Copy(clone.CpuStates, r.CpuStates)
	maps.Copy(clone.CpuStatesPercent, r.CpuStatesPercent)
	maps.Copy(clone.Other, r.Other)
	maps.Copy(clone.OtherPercent, r.OtherPercent)
	maps.Copy(clone.PkgStatesPercent, r.PkgStatesPercent)
	return clone
}

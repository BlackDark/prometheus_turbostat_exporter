package internal

import "maps"

type TurbostatRow struct {
	Pkg               string
	Core              string
	CPU               string
	CoreStatesPercent map[string]float64
	CPUStates         map[string]float64
	CPUStatesPercent  map[string]float64
	Other             map[string]float64
	OtherPercent      map[string]float64
	PkgStatesPercent  map[string]float64
	Category          string // "total", "package", "core", "cpu"
}

func NewTurbostatRow() *TurbostatRow {
	return &TurbostatRow{
		Pkg:               "0",
		CoreStatesPercent: map[string]float64{},
		CPUStates:         map[string]float64{},
		CPUStatesPercent:  map[string]float64{},
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
		CPU:               r.CPU,
		CoreStatesPercent: make(map[string]float64, len(r.CoreStatesPercent)),
		CPUStates:         make(map[string]float64, len(r.CPUStates)),
		CPUStatesPercent:  make(map[string]float64, len(r.CPUStatesPercent)),
		Other:             make(map[string]float64, len(r.Other)),
		OtherPercent:      make(map[string]float64, len(r.OtherPercent)),
		PkgStatesPercent:  make(map[string]float64, len(r.PkgStatesPercent)),
		Category:          category,
	}
	maps.Copy(clone.CoreStatesPercent, r.CoreStatesPercent)
	maps.Copy(clone.CPUStates, r.CPUStates)
	maps.Copy(clone.CPUStatesPercent, r.CPUStatesPercent)
	maps.Copy(clone.Other, r.Other)
	maps.Copy(clone.OtherPercent, r.OtherPercent)
	maps.Copy(clone.PkgStatesPercent, r.PkgStatesPercent)
	return clone
}

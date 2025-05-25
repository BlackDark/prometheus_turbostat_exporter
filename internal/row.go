package internal

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

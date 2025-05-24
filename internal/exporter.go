package internal

import (
	"maps"

	"github.com/prometheus/client_golang/prometheus"
)

type TurbostatExporter struct {
	coreStatesPercent      *prometheus.GaugeVec
	cpuStates              *prometheus.GaugeVec
	cpuStatesPercent       *prometheus.GaugeVec
	total                  *prometheus.GaugeVec
	totalCoreStatesPercent *prometheus.GaugeVec // TODO: unused
	totalCpuStates         *prometheus.GaugeVec // TODO: unused
	totalCpuStatesPercent  *prometheus.GaugeVec // TODO: unused
	totalPercent           *prometheus.GaugeVec
	totalPkgStatesPercent  *prometheus.GaugeVec
}

func NewTurbostatExporter() *TurbostatExporter {
	labels := []string{"package", "core", "cpu", "type"}

	exporter := &TurbostatExporter{
		coreStatesPercent: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "turbostats_core_states_percent",
		}, labels),
		cpuStates: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "turbostats_cpu_states",
		}, labels),
		cpuStatesPercent: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "turbostats_cpu_states_percent",
		}, labels),
		total: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "turbostats_total",
		}, labels),
		totalCoreStatesPercent: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "turbostats_total_core_states_percent",
		}, labels),
		totalCpuStates: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "turbostats_total_cpu_states",
		}, labels),
		totalCpuStatesPercent: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "turbostats_total_cpu_states_percent",
		}, labels),
		totalPercent: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "turbostats_total_percent",
		}, labels),
		totalPkgStatesPercent: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "turbostats_total_pkg_states_percent",
		}, labels),
	}
	exporter.register()

	return exporter
}

func (e *TurbostatExporter) register() {
	prometheus.MustRegister(
		e.coreStatesPercent,
		e.cpuStates,
		e.cpuStatesPercent,
		e.total,
		e.totalCoreStatesPercent,
		e.totalCpuStates,
		e.totalCpuStatesPercent,
		e.totalPercent,
		e.totalPkgStatesPercent,
	)
}

// Update uses the collected turbostat data of all TurbostatRows and configures the prometheus metrics.
func (e *TurbostatExporter) Update(rows []TurbostatRow) {
	for _, row := range rows {
		baseLabels := prometheus.Labels{}
		baseLabels["package"] = row.Pkg
		baseLabels["core"] = row.Core
		baseLabels["cpu"] = row.Cpu
		// NOTE: we could configure special metrics if e.g. package == '-'

		for t, v := range row.CpuStates {
			labels := mergeLabels(baseLabels, prometheus.Labels{"type": t})
			e.cpuStates.With(labels).Set(v)
		}

		for t, v := range row.CpuStatesPercent {
			labels := mergeLabels(baseLabels, prometheus.Labels{"type": t})
			e.cpuStatesPercent.With(labels).Set(v)
		}

		for t, v := range row.CoreStatesPercent {
			labels := mergeLabels(baseLabels, prometheus.Labels{"type": t})
			e.coreStatesPercent.With(labels).Set(v)
		}

		for t, v := range row.Other {
			labels := mergeLabels(baseLabels, prometheus.Labels{"type": t})
			e.total.With(labels).Set(v)
		}

		for t, v := range row.OtherPercent {
			labels := mergeLabels(baseLabels, prometheus.Labels{"type": t})
			e.totalPercent.With(labels).Set(v)
		}

		for t, v := range row.PkgStatesPercent {
			labels := mergeLabels(baseLabels, prometheus.Labels{"type": t})
			e.totalPkgStatesPercent.With(labels).Set(v)
		}

	}
}

func mergeLabels(a, b prometheus.Labels) prometheus.Labels {
	res := make(prometheus.Labels, len(a)+len(b))
	maps.Copy(res, a)
	maps.Copy(res, b)
	return res
}

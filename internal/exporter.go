package internal

import (
	"github.com/prometheus/client_golang/prometheus"
)

type TurbostatExporter struct {
	total           *prometheus.GaugeVec
	totalPercent    *prometheus.GaugeVec
	packages        *prometheus.GaugeVec
	packagesPercent *prometheus.GaugeVec
	cores           *prometheus.GaugeVec
	coresPercent    *prometheus.GaugeVec
	cpus            *prometheus.GaugeVec
	cpusPercent     *prometheus.GaugeVec
}

func NewTurbostatExporter() *TurbostatExporter {
	labelsTotal := []string{"type"}
	labelsPackage := append(labelsTotal, "package")
	labelsCore := append(labelsPackage, "core")
	labelsCpu := append(labelsCore, "cpu")

	exporter := &TurbostatExporter{
		packages: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "turbostat_packages",
			Help: "Metrics for the whole package",
		}, []string{"package", "type"}),
		cores: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "turbostat_cores",
		}, []string{"package", "core", "type"}),
		cpus: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "turbostat_cpus",
		}, []string{"package", "core", "cpu", "type"}),
		packagesPercent: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "turbostat_packages_percent",
		}, labelsPackage),
		coresPercent: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "turbostat_cores_percent",
		}, labelsCore),
		cpusPercent: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "turbostat_cpus_percent",
		}, labelsCpu),
		totalPercent: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "turbostat_total_percent",
			Help: "Metrics for the whole system in percentages. First line in output.",
		}, labelsTotal),
		total: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "turbostat_total",
			Help: "Metrics for the whole system. First line in output.",
		}, labelsTotal),
	}
	exporter.register()

	return exporter
}

func (e *TurbostatExporter) register() {
	prometheus.MustRegister(
		e.total,
		e.packages,
		e.cores,
		e.cpus,
		e.packagesPercent,
		e.coresPercent,
		e.cpusPercent,
		e.totalPercent,
	)
}

func (e *TurbostatExporter) resetAll() {
	e.packages.Reset()
	e.cores.Reset()
	e.cpus.Reset()
	e.packagesPercent.Reset()
	e.coresPercent.Reset()
	e.cpusPercent.Reset()
	e.total.Reset()
	e.totalPercent.Reset()
}

// Update uses the collected turbostat data of all TurbostatRows and configures the prometheus metrics.
func (e *TurbostatExporter) Update(rows []TurbostatRow) {
	e.resetAll()
	for _, row := range rows {
		switch row.Category {
		case "package":
			for t, v := range row.Other {
				e.packages.With(prometheus.Labels{"package": row.Pkg, "type": sanitizeHeader(t)}).Set(v)
			}
			for t, v := range row.OtherPercent {
				e.packagesPercent.With(prometheus.Labels{"package": row.Pkg, "type": sanitizeHeader(t)}).Set(v)
			}
		case "core":
			for t, v := range row.Other {
				e.cores.With(prometheus.Labels{"package": row.Pkg, "core": row.Core, "type": sanitizeHeader(t)}).Set(v)
			}
			for t, v := range row.OtherPercent {
				e.coresPercent.With(prometheus.Labels{"package": row.Pkg, "core": row.Core, "type": sanitizeHeader(t)}).Set(v)
			}
		case "cpu":
			for t, v := range row.Other {
				e.cpus.With(prometheus.Labels{"package": row.Pkg, "core": row.Core, "cpu": row.Cpu, "type": sanitizeHeader(t)}).Set(v)
			}
			for t, v := range row.OtherPercent {
				e.cpusPercent.With(prometheus.Labels{"package": row.Pkg, "core": row.Core, "cpu": row.Cpu, "type": sanitizeHeader(t)}).Set(v)
			}
		case "total":
			for t, v := range row.Other {
				e.total.With(prometheus.Labels{"type": sanitizeHeader(t)}).Set(v)
			}
			for t, v := range row.OtherPercent {
				e.totalPercent.With(prometheus.Labels{"type": sanitizeHeader(t)}).Set(v)
			}
		}
	}
}

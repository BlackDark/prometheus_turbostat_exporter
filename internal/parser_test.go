package internal

import (
	"os"
	"strings"
	"testing"
)

// its an i5-12500 so, it has 1 package, 14 cores, 20 cpus
func TestParseRows_SinglePackage(t *testing.T) {
	parser := NewTurbostatParser()

	content, err := os.ReadFile("../data/prox.tsv")
	if err != nil {
		t.Errorf("expected parsing prox.tsv to succeed, got error: %v", err)
		return
	}

	headers, rows, err := ParseTurbostatOutput(string(content))

	if err != nil {
		t.Errorf("expected parsing prox.tsv to succeed, got error: %v", err)
		return
	}

	categorized := parser.ParseRowsSimple(headers, rows)

	if len(categorized["total"]) != 1 {
		t.Errorf("expected 1 total row, got %d", len(categorized["total"]))
	}
	if len(categorized["package"]) != 1 {
		t.Errorf("expected 2 package row, got %d", len(categorized["package"]))
	}
	if len(categorized["core"]) != 14 {
		t.Errorf("expected 16 core rows, got %d", len(categorized["core"]))
	}
	if len(categorized["cpu"]) != 20 {
		t.Errorf("expected 32 cpu row, got %d", len(categorized["cpu"]))
	}

	// Check that percent columns are only in OtherPercent
	row := categorized["cpu"][0]
	for k := range row.OtherPercent {
		if !strings.Contains(k, "%") {
			t.Errorf("expected percent key to end with %%: %s", k)
		}
	}
	for k := range row.Other {
		if strings.Contains(k, "%") {
			t.Errorf("did not expect percent key in Other: %s", k)
		}
	}
}

func TestParseRows_SandyBridge(t *testing.T) {
	parser := NewTurbostatParser()

	content, err := os.ReadFile("../data/sandy-bridge.tsv")
	if err != nil {
		t.Errorf("expected parsing sandy-bridge.tsv to succeed, got error: %v", err)
		return
	}

	headers, rows, err := ParseTurbostatOutput(string(content))

	if err != nil {
		t.Errorf("expected parsing sandy-bridge.tsv to succeed, got error: %v", err)
		return
	}

	categorized := parser.ParseRowsSimple(headers, rows)

	if len(categorized["total"]) != 1 {
		t.Errorf("expected 1 total row, got %d", len(categorized["total"]))
	}
	if len(categorized["package"]) != 2 {
		t.Errorf("expected 2 package row, got %d", len(categorized["package"]))
	}
	if len(categorized["core"]) != 16 {
		t.Errorf("expected 16 core rows, got %d", len(categorized["core"]))
	}
	if len(categorized["cpu"]) != 32 {
		t.Errorf("expected 32 cpu row, got %d", len(categorized["cpu"]))
	}

	// Check that percent columns are only in OtherPercent
	row := categorized["cpu"][0]
	for k := range row.OtherPercent {
		if !strings.Contains(k, "%") {
			t.Errorf("expected percent key to end with %%: %s", k)
		}
	}
	for k := range row.Other {
		if strings.Contains(k, "%") {
			t.Errorf("did not expect percent key in Other: %s", k)
		}
	}
}

func TestParseRows_Empty(t *testing.T) {
	parser := NewTurbostatParser()

	categorized := parser.ParseRowsSimple([]string{}, [][]string{})
	for _, cat := range []string{"total", "package", "core", "cpu"} {
		if len(categorized[cat]) != 0 {
			t.Errorf("expected 0 rows for %s, got %d", cat, len(categorized[cat]))
		}
	}
}

func TestParseRows_SingleRow(t *testing.T) {
	parser := NewTurbostatParser()

	headers := []string{"Package", "Core", "CPU", "Bzy_MHz"}
	rows := [][]string{{"0", "0", "0", "2000"}}
	categorized := parser.ParseRowsSimple(headers, rows)
	if len(categorized["cpu"]) != 1 {
		t.Errorf("expected 1 cpu row, got %d", len(categorized["cpu"]))
	}
}

// Reproduces https://github.com/BlackDark/prometheus_turbostat_exporter/issues/11
// When turbostat lacks sufficient privileges (e.g. no root/CAP_SYS_RAWIO), it prints
// a warning sentence on stdout/stderr before the real header line. main.go merges
// stdout+stderr, so that sentence must not be mistaken for the turbostat header.
func TestParseOutput_PrivilegeWarningBeforeHeader(t *testing.T) {
	headers, rows, err := ParseTurbostatOutput(`Some of the counters may not be available to access /dev/cpu/0/msr.
Core	CPU	Avg_MHz	Busy%
-	-	125	6.57
0	0	80	9.91
`)

	if err != nil {
		t.Fatalf("expected parsing turbostat output to succeed, got error: %v", err)
	}

	wantHeaders := []string{"Core", "CPU", "Avg_MHz", "Busy%"}
	if len(headers) != len(wantHeaders) {
		t.Fatalf("expected headers %v, got %v", wantHeaders, headers)
	}
	for i, h := range wantHeaders {
		if headers[i] != h {
			t.Errorf("expected header[%d] = %q, got %q", i, h, headers[i])
		}
	}

	if len(rows) != 2 {
		t.Fatalf("expected 2 data rows, got %d: %v", len(rows), rows)
	}
}

func TestParseOutput_RealWithSeconds(t *testing.T) {
	headers, rows, err := ParseTurbostatOutput(`0.007923 sec
Core CPU Test
-	-	1
0	0	1
`)

	if err != nil {
		t.Errorf("expected parsing turbostat output to succeed, got error: %v", err)
	}

	if len(headers) == 0 || len(rows) == 0 {
		t.Errorf("expected non-empty headers and rows, got %d headers and %d rows", len(headers), len(rows))
		return
	}

	if len(headers) != 3 {
		t.Errorf("expected 3 headers, got %d", len(headers))
	}
}

func parseFixture(t *testing.T, path string) map[string][]*TurbostatRow {
	t.Helper()
	content, err := os.ReadFile(path) //nolint:gosec // fixed test fixture paths
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	headers, rows, err := ParseTurbostatOutput(string(content))
	if err != nil {
		t.Fatalf("parsing %s: %v", path, err)
	}
	return NewTurbostatParser().ParseRowsSimple(headers, rows)
}

func findRow(rows []*TurbostatRow, cpu string) *TurbostatRow {
	for _, r := range rows {
		if r.CPU == cpu {
			return r
		}
	}
	return nil
}

func assertNoTopologyMetrics(t *testing.T, categorized map[string][]*TurbostatRow) {
	t.Helper()
	for cat, rows := range categorized {
		for _, r := range rows {
			for h := range topologyColumns {
				if _, ok := r.Other[h]; ok {
					t.Errorf("%s row exposes topology column %q as metric", cat, h)
				}
			}
		}
	}
}

// Pre-2026 turbostat: no Module column, decimal Core IDs.
func TestParseRows_LegacyLabels(t *testing.T) {
	categorized := parseFixture(t, "../data/prox.tsv")

	cpu := findRow(categorized["cpu"], "13")
	if cpu == nil {
		t.Fatal("expected cpu 13 row")
	}
	if cpu.Pkg != "0" || cpu.Core != "25" {
		t.Errorf("expected package 0 core 25, got package %q core %q", cpu.Pkg, cpu.Core)
	}
	if cpu.Other["Avg_MHz"] != 26 {
		t.Errorf("expected Avg_MHz 26, got %v", cpu.Other["Avg_MHz"])
	}
	assertNoTopologyMetrics(t, categorized)
}

// turbostat 2026.04.21+: leading Module column, Core/Module IDs in hex.
func TestParseRows_ModuleColumn(t *testing.T) {
	categorized := parseFixture(t, "../data/turbostat-2026-module.tsv")

	wantCounts := map[string]int{"total": 1, "package": 1, "core": 6, "cpu": 12}
	for cat, want := range wantCounts {
		if got := len(categorized[cat]); got != want {
			t.Errorf("expected %d %s rows, got %d", want, cat, got)
		}
	}

	cpu := findRow(categorized["cpu"], "11")
	if cpu == nil {
		t.Fatal("expected cpu 11 row")
	}
	if cpu.Pkg != "0" || cpu.Core != "5" {
		t.Errorf("expected package 0 core 5, got package %q core %q", cpu.Pkg, cpu.Core)
	}
	if cpu.Other["Avg_MHz"] != 17 {
		t.Errorf("expected Avg_MHz 17, got %v", cpu.Other["Avg_MHz"])
	}
	if cpu.OtherPercent["LLC%hit"] != 40.18 {
		t.Errorf("expected LLC%%hit 40.18, got %v", cpu.OtherPercent["LLC%hit"])
	}

	total := categorized["total"][0]
	if total.Other["Avg_MHz"] != 32 {
		t.Errorf("expected total Avg_MHz 32, got %v", total.Other["Avg_MHz"])
	}
	for _, cat := range []string{"total", "package"} {
		if v, ok := categorized[cat][0].OtherPercent["GFX%rc6"]; ok {
			t.Errorf("expected nan GFX%%rc6 to be skipped in %s row, got %v", cat, v)
		}
	}
	assertNoTopologyMetrics(t, categorized)
}

func TestParseRows_MultiPackageModuleColumn(t *testing.T) {
	headers, rows, err := ParseTurbostatOutput(`Package	Module	Core	CPU	Avg_MHz	Busy%	CPU%c6	CoreTmp	PkgTmp
-	-	-	-	10	1.00	90.00	30	40
0	0x0	0x0	0	20	2.00	91.00	31	41
0	0x0	0x0	4	21	2.10
0	0x2	0xa	1	22	2.20	92.00	32
1	0x4	0x0	2	23	2.30	93.00	33	42
1	0x4	0x0	3	24	2.40
`)
	if err != nil {
		t.Fatalf("parsing: %v", err)
	}
	categorized := NewTurbostatParser().ParseRowsSimple(headers, rows)

	if got := len(categorized["package"]); got != 2 {
		t.Fatalf("expected 2 package rows, got %d", got)
	}
	if categorized["package"][1].Pkg != "1" {
		t.Errorf("expected second package id 1, got %q", categorized["package"][1].Pkg)
	}

	cpu := findRow(categorized["cpu"], "1")
	if cpu == nil {
		t.Fatal("expected cpu 1 row")
	}
	if cpu.Pkg != "0" || cpu.Core != "10" {
		t.Errorf("expected package 0 core 10, got package %q core %q", cpu.Pkg, cpu.Core)
	}
	if cpu.Other["Avg_MHz"] != 22 {
		t.Errorf("expected Avg_MHz 22, got %v", cpu.Other["Avg_MHz"])
	}
	assertNoTopologyMetrics(t, categorized)
}

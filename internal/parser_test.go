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

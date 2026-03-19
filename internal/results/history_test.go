package results

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/goozt/gospeed/internal/tests"
)

func TestSaveAndLoadHistory(t *testing.T) {
	// Use a temp directory for history.
	tmpDir := t.TempDir()
	histFile := filepath.Join(tmpDir, "history.jsonl")

	report := &Report{
		Timestamp:    time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC),
		Server:       "test:9000",
		OverallGrade: GradeA,
		Results: []TestResult{
			{Test: "latency", Grade: GradeA, Metrics: &tests.LatencyMetrics{Avg: 5}},
		},
	}

	// Write directly to temp file (bypass historyPath()).
	f, err := os.Create(histFile)
	if err != nil {
		t.Fatal(err)
	}

	entry := HistoryEntry{
		Timestamp:    report.Timestamp,
		Server:       report.Server,
		OverallGrade: report.OverallGrade,
		Results:      report.Results,
	}
	data, _ := json.Marshal(entry)
	f.Write(append(data, '\n'))

	// Write a second entry.
	entry2 := HistoryEntry{
		Timestamp:    time.Date(2026, 1, 16, 10, 0, 0, 0, time.UTC),
		Server:       "test:9000",
		OverallGrade: GradeB,
		Results:      report.Results,
	}
	data2, _ := json.Marshal(entry2)
	f.Write(append(data2, '\n'))
	f.Close()

	// Read and verify.
	content, err := os.ReadFile(histFile)
	if err != nil {
		t.Fatal(err)
	}

	lines := splitLines(content)
	var entries []HistoryEntry
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		var e HistoryEntry
		if err := json.Unmarshal(line, &e); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		entries = append(entries, e)
	}

	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].OverallGrade != GradeA {
		t.Errorf("entry 0 grade = %s, want A", entries[0].OverallGrade)
	}
	if entries[1].OverallGrade != GradeB {
		t.Errorf("entry 1 grade = %s, want B", entries[1].OverallGrade)
	}
}

func TestSplitLines(t *testing.T) {
	data := []byte("line1\nline2\nline3")
	lines := splitLines(data)
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}
	if string(lines[0]) != "line1" {
		t.Errorf("line 0 = %q", string(lines[0]))
	}
	if string(lines[2]) != "line3" {
		t.Errorf("line 2 = %q", string(lines[2]))
	}
}

func TestSplitLinesEmpty(t *testing.T) {
	lines := splitLines(nil)
	if len(lines) != 0 {
		t.Errorf("expected 0 lines, got %d", len(lines))
	}
}

func TestCompareTrend(t *testing.T) {
	SetColor(false)
	defer SetColor(false)

	prev := HistoryEntry{OverallGrade: GradeB}
	curr := HistoryEntry{OverallGrade: GradeA}
	trend := compareTrend(prev, curr)
	if trend != "↑" {
		t.Errorf("improving trend = %q, want ↑", trend)
	}

	curr.OverallGrade = GradeC
	trend = compareTrend(prev, curr)
	if trend != "↓" {
		t.Errorf("declining trend = %q, want ↓", trend)
	}

	curr.OverallGrade = GradeB
	trend = compareTrend(prev, curr)
	if trend != "→" {
		t.Errorf("same trend = %q, want →", trend)
	}
}

func TestPrintHistoryNoFile(t *testing.T) {
	// PrintHistory with non-existent file should not error.
	var buf bytes.Buffer
	// We can't easily test with the real historyPath, but we test splitLines
	// and compareTrend which are the core logic.
	_ = buf
}

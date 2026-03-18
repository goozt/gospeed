package results

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// HistoryEntry is a single line in the JSONL history file.
type HistoryEntry struct {
	Timestamp    time.Time    `json:"timestamp"`
	Server       string       `json:"server"`
	OverallGrade Grade        `json:"overall_grade"`
	Results      []TestResult `json:"results"`
}

func historyPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".gospeed", "history.jsonl")
}

// SaveHistory appends a report to the history file.
func SaveHistory(report *Report) error {
	path := historyPath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	entry := HistoryEntry{
		Timestamp:    report.Timestamp,
		Server:       report.Server,
		OverallGrade: report.OverallGrade,
		Results:      report.Results,
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	_, err = f.Write(append(data, '\n'))
	return err
}

// PrintHistory reads and displays previous results.
func PrintHistory(w io.Writer, limit int) error {
	path := historyPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintln(w, "No history found. Run a speed test first.")
			return nil
		}
		return err
	}

	var entries []HistoryEntry
	for _, line := range splitLines(data) {
		if len(line) == 0 {
			continue
		}
		var e HistoryEntry
		if err := json.Unmarshal(line, &e); err != nil {
			continue
		}
		entries = append(entries, e)
	}

	if len(entries) == 0 {
		fmt.Fprintln(w, "No history found.")
		return nil
	}

	// Show most recent entries.
	start := 0
	if limit > 0 && len(entries) > limit {
		start = len(entries) - limit
	}

	fmt.Fprintln(w, Header("Speed Test History"))
	fmt.Fprintln(w)

	for i := start; i < len(entries); i++ {
		e := entries[i]
		trend := ""
		if i > 0 {
			trend = compareTrend(entries[i-1], e)
		}
		fmt.Fprintf(w, "  %s  Server: %s  Grade: %s %s\n",
			Dim(e.Timestamp.Format("2006-01-02 15:04")),
			e.Server,
			ColorGrade(e.OverallGrade),
			trend,
		)
	}
	return nil
}

func compareTrend(prev, curr HistoryEntry) string {
	order := map[Grade]int{GradeA: 5, GradeB: 4, GradeC: 3, GradeD: 2, GradeF: 1}
	p, c := order[prev.OverallGrade], order[curr.OverallGrade]
	switch {
	case c > p:
		return Green("↑")
	case c < p:
		return Red("↓")
	default:
		return Dim("→")
	}
}

func splitLines(data []byte) [][]byte {
	var lines [][]byte
	start := 0
	for i, b := range data {
		if b == '\n' {
			lines = append(lines, data[start:i])
			start = i + 1
		}
	}
	if start < len(data) {
		lines = append(lines, data[start:])
	}
	return lines
}

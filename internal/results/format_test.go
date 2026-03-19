package results

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/goozt/gospeed/internal/tests"
)

func init() {
	// Disable colors for predictable test output.
	SetColor(false)
}

func sampleReport() *Report {
	return &Report{
		Timestamp: time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC),
		Server:    "test-server:9000",
		Results: []TestResult{
			{Test: "latency", Grade: GradeA, Metrics: &tests.LatencyMetrics{
				Samples: []float64{1, 2, 3}, Min: 1, Max: 3, Avg: 2, Median: 2, P95: 3, StdDev: 0.82,
			}},
			{Test: "tcp", Grade: GradeB, Metrics: &tests.TCPMetrics{
				Direction: "upload", Duration: 10, BytesTotal: 125000000, BitsPerSec: 100_000_000, Streams: 4,
			}},
			{Test: "udp", Grade: GradeA, Metrics: &tests.UDPMetrics{
				Duration: 10, BytesTotal: 100000000, BitsPerSec: 80_000_000,
				PacketsSent: 10000, PacketsRecv: 9990, PacketsLost: 10, LossPercent: 0.1,
			}},
			{Test: "jitter", Grade: GradeA, Metrics: &tests.JitterMetrics{
				AvgJitter: 1.5, MinJitter: 0.5, MaxJitter: 3.0, StdDev: 0.7, PacketsRecv: 200, PacketsSent: 200,
			}},
			{Test: "mtu", Metrics: &tests.MTUMetrics{MTU: 1500}},
			{Test: "bufferbloat", Grade: GradeA, Metrics: &tests.BufferbloatMetrics{
				UnloadedLatency: tests.LatencyMetrics{Avg: 2},
				LoadedLatency:   tests.LatencyMetrics{Avg: 5},
				RPM:             12000, LatencyIncrease: 3,
				Throughput: tests.TCPMetrics{Direction: "upload", BitsPerSec: 100_000_000},
			}},
			{Test: "dns", Metrics: &tests.DNSMetrics{Host: "example.com", Min: 1, Max: 5, Avg: 2.5}},
			{Test: "connect", Metrics: &tests.ConnectMetrics{Min: 0.5, Max: 2, Avg: 1.2}},
			{Test: "bidir", Metrics: &tests.BidirMetrics{
				Upload:   tests.TCPMetrics{BitsPerSec: 100_000_000},
				Download: tests.TCPMetrics{BitsPerSec: 200_000_000},
			}},
		},
		OverallGrade: GradeB,
	}
}

func TestFormatTable(t *testing.T) {
	var buf bytes.Buffer
	FormatTable(&buf, sampleReport())
	out := buf.String()

	// Check key sections are present.
	checks := []string{
		"test-server:9000",
		"Latency",
		"TCP Throughput",
		"UDP Throughput",
		"Jitter",
		"Path MTU",
		"Bufferbloat",
		"DNS Resolution",
		"TCP Connect Time",
		"Bidirectional",
		"Overall",
	}
	for _, c := range checks {
		if !strings.Contains(out, c) {
			t.Errorf("table output missing %q", c)
		}
	}
}

func TestFormatJSON(t *testing.T) {
	var buf bytes.Buffer
	if err := FormatJSON(&buf, sampleReport()); err != nil {
		t.Fatalf("FormatJSON: %v", err)
	}

	// Verify it's valid JSON.
	var parsed map[string]any
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	// Check key fields.
	if _, ok := parsed["timestamp"]; !ok {
		t.Error("missing timestamp")
	}
	if _, ok := parsed["server"]; !ok {
		t.Error("missing server")
	}
	if _, ok := parsed["results"]; !ok {
		t.Error("missing results")
	}
	if _, ok := parsed["overall_grade"]; !ok {
		t.Error("missing overall_grade")
	}
}

func TestFormatCSV(t *testing.T) {
	var buf bytes.Buffer
	FormatCSV(&buf, sampleReport())
	out := buf.String()

	lines := strings.Split(strings.TrimSpace(out), "\n")

	// Header line.
	if lines[0] != "timestamp,server,test,grade,metric,value" {
		t.Errorf("CSV header = %q", lines[0])
	}

	// Should have multiple data lines.
	if len(lines) < 5 {
		t.Errorf("expected >= 5 CSV lines, got %d", len(lines))
	}

	// Check some content.
	if !strings.Contains(out, "latency") {
		t.Error("CSV missing latency")
	}
	if !strings.Contains(out, "tcp") {
		t.Error("CSV missing tcp")
	}
	if !strings.Contains(out, "udp") {
		t.Error("CSV missing udp")
	}
}

func TestFormatBPS(t *testing.T) {
	tests := []struct {
		bps  float64
		want string
	}{
		{500, "500 bps"},
		{1500, "1.50 Kbps"},
		{50_000_000, "50.00 Mbps"},
		{1_500_000_000, "1.50 Gbps"},
	}
	for _, tt := range tests {
		got := formatBPS(tt.bps)
		if got != tt.want {
			t.Errorf("formatBPS(%f) = %q, want %q", tt.bps, got, tt.want)
		}
	}
}

func TestComputeOverallGradeVariants(t *testing.T) {
	tests := []struct {
		grades []Grade
		want   Grade
	}{
		{[]Grade{GradeA, GradeA}, GradeA},
		{[]Grade{GradeA, GradeB}, GradeB},
		{[]Grade{GradeA, GradeF}, GradeF},
		{[]Grade{GradeC, GradeD}, GradeD},
		{nil, GradeA},
	}
	for _, tt := range tests {
		results := make([]TestResult, len(tt.grades))
		for i, g := range tt.grades {
			results[i] = TestResult{Grade: g}
		}
		got := ComputeOverallGrade(results)
		if got != tt.want {
			t.Errorf("ComputeOverallGrade(%v) = %s, want %s", tt.grades, got, tt.want)
		}
	}
}

func TestColorGradeNoColor(t *testing.T) {
	SetColor(false)
	defer SetColor(false)

	for _, g := range []Grade{GradeA, GradeB, GradeC, GradeD, GradeF} {
		got := ColorGrade(g)
		if got != string(g) {
			t.Errorf("ColorGrade(%s) with color disabled = %q, want %q", g, got, string(g))
		}
	}
}

func TestColorGradeWithColor(t *testing.T) {
	SetColor(true)
	defer SetColor(false)

	for _, g := range []Grade{GradeA, GradeB, GradeC, GradeD, GradeF} {
		got := ColorGrade(g)
		if !strings.Contains(got, string(g)) {
			t.Errorf("ColorGrade(%s) should contain grade letter", g)
		}
		if !strings.Contains(got, "\033[") {
			t.Errorf("ColorGrade(%s) should contain ANSI escape", g)
		}
	}
}

func TestHeaderDimBold(t *testing.T) {
	SetColor(false)
	defer SetColor(false)

	h := Header("test")
	if !strings.Contains(h, "test") {
		t.Error("Header should contain text")
	}

	d := Dim("grey")
	if d != "grey" {
		t.Errorf("Dim with no color = %q, want 'grey'", d)
	}

	b := Bold("bold")
	if b != "bold" {
		t.Errorf("Bold with no color = %q, want 'bold'", b)
	}
}

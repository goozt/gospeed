package results

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/goozt/gospeed/internal/tests"
)

// TestResult holds the result of a single test with its type and metrics.
type TestResult struct {
	Test    string `json:"test"`
	Metrics any    `json:"metrics"`
	Grade   Grade  `json:"grade,omitempty"`
}

// Report holds the complete results of a speed test run.
type Report struct {
	Timestamp    time.Time    `json:"timestamp"`
	Server       string       `json:"server"`
	Results      []TestResult `json:"results"`
	OverallGrade Grade        `json:"overall_grade"`
}

// FormatTable writes a human-readable table to w.
func FormatTable(w io.Writer, report *Report) {
	fmt.Fprintln(w, Header("gospeed results"))
	fmt.Fprintf(w, "  Server: %s\n", Bold(report.Server))
	fmt.Fprintf(w, "  Time:   %s\n\n", Dim(report.Timestamp.Format(time.RFC3339)))

	for _, r := range report.Results {
		formatTestResult(w, r)
		fmt.Fprintln(w)
	}

	fmt.Fprintf(w, "%s\n", Header(fmt.Sprintf("Overall: %s", ColorGrade(report.OverallGrade))))
}

func formatTestResult(w io.Writer, r TestResult) {
	switch r.Test {
	case "latency":
		m, ok := r.Metrics.(*tests.LatencyMetrics)
		if !ok {
			return
		}
		fmt.Fprintf(w, "  %s %s\n", Bold("Latency"), ColorGrade(r.Grade))
		fmt.Fprintf(w, "    Min: %.2f ms  Avg: %.2f ms  Max: %.2f ms\n", m.Min, m.Avg, m.Max)
		fmt.Fprintf(w, "    Median: %.2f ms  P95: %.2f ms  StdDev: %.2f ms\n", m.Median, m.P95, m.StdDev)

	case "mtu":
		m, ok := r.Metrics.(*tests.MTUMetrics)
		if !ok {
			return
		}
		fmt.Fprintf(w, "  %s\n", Bold("Path MTU"))
		fmt.Fprintf(w, "    MTU: %d bytes\n", m.MTU)

	case "tcp":
		m, ok := r.Metrics.(*tests.TCPMetrics)
		if !ok {
			return
		}
		fmt.Fprintf(w, "  %s (%s) %s\n", Bold("TCP Throughput"), m.Direction, ColorGrade(r.Grade))
		fmt.Fprintf(w, "    Speed: %s  (%d streams, %.1fs)\n",
			formatBPS(m.BitsPerSec), m.Streams, m.Duration)

	case "udp":
		m, ok := r.Metrics.(*tests.UDPMetrics)
		if !ok {
			return
		}
		fmt.Fprintf(w, "  %s %s\n", Bold("UDP Throughput"), ColorGrade(r.Grade))
		fmt.Fprintf(w, "    Speed: %s  Loss: %.2f%% (%d/%d packets)\n",
			formatBPS(m.BitsPerSec), m.LossPercent, m.PacketsLost, m.PacketsSent)
		if m.OutOfOrder > 0 {
			fmt.Fprintf(w, "    Out-of-order: %d packets\n", m.OutOfOrder)
		}

	case "jitter":
		m, ok := r.Metrics.(*tests.JitterMetrics)
		if !ok {
			return
		}
		fmt.Fprintf(w, "  %s %s\n", Bold("Jitter"), ColorGrade(r.Grade))
		fmt.Fprintf(w, "    Avg: %.2f ms  Min: %.2f ms  Max: %.2f ms  StdDev: %.2f ms\n",
			m.AvgJitter, m.MinJitter, m.MaxJitter, m.StdDev)
		fmt.Fprintf(w, "    Packets: %d/%d received\n", m.PacketsRecv, m.PacketsSent)

	case "bufferbloat":
		m, ok := r.Metrics.(*tests.BufferbloatMetrics)
		if !ok {
			return
		}
		fmt.Fprintf(w, "  %s %s\n", Bold("Bufferbloat"), ColorGrade(r.Grade))
		fmt.Fprintf(w, "    Unloaded latency: %.2f ms  Loaded latency: %.2f ms\n",
			m.UnloadedLatency.Avg, m.LoadedLatency.Avg)
		rpmStr := fmt.Sprintf("%.0f", m.RPM)
		if m.RPM >= 999999 {
			rpmStr = "∞ (unmeasurable)"
		}
		fmt.Fprintf(w, "    Latency increase: %.2f ms  RPM: %s\n",
			m.LatencyIncrease, rpmStr)
		fmt.Fprintf(w, "    Throughput during test: %s\n", formatBPS(m.Throughput.BitsPerSec))

	case "dns":
		m, ok := r.Metrics.(*tests.DNSMetrics)
		if !ok {
			return
		}
		fmt.Fprintf(w, "  %s\n", Bold("DNS Resolution"))
		fmt.Fprintf(w, "    Host: %s\n", m.Host)
		fmt.Fprintf(w, "    Min: %.2f ms  Avg: %.2f ms  Max: %.2f ms\n", m.Min, m.Avg, m.Max)

	case "connect":
		m, ok := r.Metrics.(*tests.ConnectMetrics)
		if !ok {
			return
		}
		fmt.Fprintf(w, "  %s\n", Bold("TCP Connect Time"))
		fmt.Fprintf(w, "    Min: %.2f ms  Avg: %.2f ms  Max: %.2f ms\n", m.Min, m.Avg, m.Max)

	case "bidir":
		m, ok := r.Metrics.(*tests.BidirMetrics)
		if !ok {
			return
		}
		fmt.Fprintf(w, "  %s\n", Bold("Bidirectional Throughput"))
		fmt.Fprintf(w, "    Upload:   %s\n", formatBPS(m.Upload.BitsPerSec))
		fmt.Fprintf(w, "    Download: %s\n", formatBPS(m.Download.BitsPerSec))
	}
}

// FormatJSON writes results as JSON to w.
func FormatJSON(w io.Writer, report *Report) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(report)
}

// FormatCSV writes results as CSV to w.
func FormatCSV(w io.Writer, report *Report) {
	fmt.Fprintln(w, "timestamp,server,test,grade,metric,value")
	ts := report.Timestamp.Format(time.RFC3339)
	for _, r := range report.Results {
		writeCSVMetrics(w, ts, report.Server, r)
	}
}

func writeCSVMetrics(w io.Writer, ts, server string, r TestResult) {
	switch m := r.Metrics.(type) {
	case *tests.LatencyMetrics:
		fmt.Fprintf(w, "%s,%s,%s,%s,avg_ms,%.2f\n", ts, server, r.Test, r.Grade, m.Avg)
		fmt.Fprintf(w, "%s,%s,%s,%s,min_ms,%.2f\n", ts, server, r.Test, r.Grade, m.Min)
		fmt.Fprintf(w, "%s,%s,%s,%s,max_ms,%.2f\n", ts, server, r.Test, r.Grade, m.Max)
	case *tests.TCPMetrics:
		fmt.Fprintf(w, "%s,%s,%s,%s,bits_per_sec,%.0f\n", ts, server, r.Test, r.Grade, m.BitsPerSec)
	case *tests.UDPMetrics:
		fmt.Fprintf(w, "%s,%s,%s,%s,bits_per_sec,%.0f\n", ts, server, r.Test, r.Grade, m.BitsPerSec)
		fmt.Fprintf(w, "%s,%s,%s,%s,loss_percent,%.2f\n", ts, server, r.Test, r.Grade, m.LossPercent)
	case *tests.JitterMetrics:
		fmt.Fprintf(w, "%s,%s,%s,%s,avg_jitter_ms,%.2f\n", ts, server, r.Test, r.Grade, m.AvgJitter)
	case *tests.BufferbloatMetrics:
		fmt.Fprintf(w, "%s,%s,%s,%s,rpm,%.0f\n", ts, server, r.Test, r.Grade, m.RPM)
	case *tests.MTUMetrics:
		fmt.Fprintf(w, "%s,%s,%s,%s,mtu,%d\n", ts, server, r.Test, r.Grade, m.MTU)
	case *tests.DNSMetrics:
		fmt.Fprintf(w, "%s,%s,%s,%s,avg_ms,%.2f\n", ts, server, r.Test, r.Grade, m.Avg)
	case *tests.ConnectMetrics:
		fmt.Fprintf(w, "%s,%s,%s,%s,avg_ms,%.2f\n", ts, server, r.Test, r.Grade, m.Avg)
	}
}

func formatBPS(bps float64) string {
	switch {
	case bps >= 1e9:
		return fmt.Sprintf("%.2f Gbps", bps/1e9)
	case bps >= 1e6:
		return fmt.Sprintf("%.2f Mbps", bps/1e6)
	case bps >= 1e3:
		return fmt.Sprintf("%.2f Kbps", bps/1e3)
	default:
		return fmt.Sprintf("%.0f bps", bps)
	}
}

// ComputeOverallGrade returns the worst grade from all test results.
func ComputeOverallGrade(results []TestResult) Grade {
	grades := []Grade{GradeA}
	for _, r := range results {
		if r.Grade != "" {
			grades = append(grades, r.Grade)
		}
	}
	worst := GradeA
	order := "ABCDF"
	for _, g := range grades {
		if strings.Index(order, string(g)) > strings.Index(order, string(worst)) {
			worst = g
		}
	}
	return worst
}

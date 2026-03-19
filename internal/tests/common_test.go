package tests

import (
	"testing"
	"time"
)

func TestEncodeDecodeDataHeader(t *testing.T) {
	buf := make([]byte, 16)
	seq := uint64(42)

	before := time.Now().UnixNano()
	EncodeDataHeader(buf, seq)
	after := time.Now().UnixNano()

	hdr := DecodeDataHeader(buf)
	if hdr.Seq != seq {
		t.Errorf("seq = %d, want %d", hdr.Seq, seq)
	}
	if hdr.Timestamp < before || hdr.Timestamp > after {
		t.Errorf("timestamp %d not in range [%d, %d]", hdr.Timestamp, before, after)
	}
}

func TestEncodeDecodeDataHeaderMaxSeq(t *testing.T) {
	buf := make([]byte, 16)
	seq := uint64(^uint64(0)) // max uint64

	EncodeDataHeader(buf, seq)
	hdr := DecodeDataHeader(buf)
	if hdr.Seq != seq {
		t.Errorf("seq = %d, want %d", hdr.Seq, seq)
	}
}

func TestEncodeDecodeDataHeaderZero(t *testing.T) {
	buf := make([]byte, 16)
	EncodeDataHeader(buf, 0)
	hdr := DecodeDataHeader(buf)
	if hdr.Seq != 0 {
		t.Errorf("seq = %d, want 0", hdr.Seq)
	}
}

func TestComputeLatencyMetrics(t *testing.T) {
	samples := []float64{10, 20, 30, 40, 50}
	m := computeLatencyMetrics(samples)

	if m.Min != 10 {
		t.Errorf("min = %f, want 10", m.Min)
	}
	if m.Max != 50 {
		t.Errorf("max = %f, want 50", m.Max)
	}
	if m.Avg != 30 {
		t.Errorf("avg = %f, want 30", m.Avg)
	}
	if m.Median != 30 {
		t.Errorf("median = %f, want 30", m.Median)
	}
}

func TestComputeLatencyMetricsEmpty(t *testing.T) {
	m := computeLatencyMetrics(nil)
	if m.Min != 0 || m.Max != 0 || m.Avg != 0 {
		t.Errorf("empty metrics should be zero: %+v", m)
	}
}

func TestComputeLatencyMetricsSingle(t *testing.T) {
	m := computeLatencyMetrics([]float64{5.5})
	if m.Min != 5.5 || m.Max != 5.5 || m.Avg != 5.5 {
		t.Errorf("single sample metrics wrong: %+v", m)
	}
}

func TestDirString(t *testing.T) {
	if dirString(false) != "upload" {
		t.Error("dirString(false) should be upload")
	}
	if dirString(true) != "download" {
		t.Error("dirString(true) should be download")
	}
}

func TestItoa(t *testing.T) {
	tests := []struct {
		in   int
		want string
	}{
		{0, "0"},
		{1, "1"},
		{42, "42"},
		{9000, "9000"},
		{12345, "12345"},
	}
	for _, tt := range tests {
		got := itoa(tt.in)
		if got != tt.want {
			t.Errorf("itoa(%d) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

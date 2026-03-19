package tests

import (
	"encoding/binary"
	"time"
)

// DataHeader is the 16-byte header prepended to every data packet.
// [8-byte sequence number][8-byte unix nanosecond timestamp]
type DataHeader struct {
	Seq       uint64
	Timestamp int64 // time.Now().UnixNano()
}

// EncodeDataHeader writes a header into buf (must be >= 16 bytes).
func EncodeDataHeader(buf []byte, seq uint64) {
	binary.BigEndian.PutUint64(buf[0:8], seq)
	binary.BigEndian.PutUint64(buf[8:16], uint64(time.Now().UnixNano()))
}

// DecodeDataHeader reads a header from buf.
func DecodeDataHeader(buf []byte) DataHeader {
	return DataHeader{
		Seq:       binary.BigEndian.Uint64(buf[0:8]),
		Timestamp: int64(binary.BigEndian.Uint64(buf[8:16])),
	}
}

// LatencyMetrics holds results from a latency test.
type LatencyMetrics struct {
	Samples []float64 `json:"samples_ms"`
	Min     float64   `json:"min_ms"`
	Max     float64   `json:"max_ms"`
	Avg     float64   `json:"avg_ms"`
	Median  float64   `json:"median_ms"`
	P95     float64   `json:"p95_ms"`
	StdDev  float64   `json:"stddev_ms"`
}

// TCPMetrics holds TCP throughput results.
type TCPMetrics struct {
	Direction  string          `json:"direction"` // "upload" or "download"
	Duration   float64         `json:"duration_s"`
	BytesTotal int64           `json:"bytes_total"`
	BitsPerSec float64         `json:"bits_per_sec"`
	Streams    int             `json:"streams"`
	Intervals  []IntervalStats `json:"intervals,omitempty"`
}

// IntervalStats holds per-second throughput data.
type IntervalStats struct {
	Start      float64 `json:"start_s"`
	End        float64 `json:"end_s"`
	Bytes      int64   `json:"bytes"`
	BitsPerSec float64 `json:"bits_per_sec"`
}

// UDPMetrics holds UDP throughput and loss results.
type UDPMetrics struct {
	Duration    float64 `json:"duration_s"`
	BytesTotal  int64   `json:"bytes_total"`
	BitsPerSec  float64 `json:"bits_per_sec"`
	PacketsSent int64   `json:"packets_sent"`
	PacketsRecv int64   `json:"packets_recv"`
	PacketsLost int64   `json:"packets_lost"`
	LossPercent float64 `json:"loss_percent"`
	OutOfOrder  int64   `json:"out_of_order"`
}

// JitterMetrics holds jitter measurement results.
type JitterMetrics struct {
	Samples     []float64 `json:"samples_ms,omitempty"`
	AvgJitter   float64   `json:"avg_jitter_ms"`
	MaxJitter   float64   `json:"max_jitter_ms"`
	MinJitter   float64   `json:"min_jitter_ms"`
	StdDev      float64   `json:"stddev_ms"`
	PacketsRecv int       `json:"packets_recv"`
	PacketsSent int       `json:"packets_sent"`
}

// MTUMetrics holds path MTU discovery results.
type MTUMetrics struct {
	MTU int `json:"mtu"`
}

// BufferbloatMetrics holds bufferbloat detection results.
type BufferbloatMetrics struct {
	UnloadedLatency LatencyMetrics `json:"unloaded_latency"`
	LoadedLatency   LatencyMetrics `json:"loaded_latency"`
	RPM             float64        `json:"rpm"` // round-trips per minute
	LatencyIncrease float64        `json:"latency_increase_ms"`
	Throughput      TCPMetrics     `json:"throughput"`
}

// DNSMetrics holds DNS resolution timing results.
type DNSMetrics struct {
	Host    string    `json:"host"`
	Samples []float64 `json:"samples_ms"`
	Min     float64   `json:"min_ms"`
	Max     float64   `json:"max_ms"`
	Avg     float64   `json:"avg_ms"`
}

// ConnectMetrics holds TCP connection setup timing results.
type ConnectMetrics struct {
	Samples []float64 `json:"samples_ms"`
	Min     float64   `json:"min_ms"`
	Max     float64   `json:"max_ms"`
	Avg     float64   `json:"avg_ms"`
}

// BidirMetrics holds bidirectional throughput results.
type BidirMetrics struct {
	Upload   TCPMetrics `json:"upload"`
	Download TCPMetrics `json:"download"`
}

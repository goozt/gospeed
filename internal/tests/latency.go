package tests

import (
	"context"
	"encoding/json"
	"math"
	"net"
	"sort"
	"time"

	"github.com/goozt/gospeed/internal/protocol"
	"github.com/goozt/gospeed/internal/server"
)

func init() {
	server.RegisterHandler(protocol.TestLatency, handleLatencyServer)
}

// latency server: echo back pings on control connection.
func handleLatencyServer(_ context.Context, conn net.Conn, params json.RawMessage) (any, error) {
	var p protocol.LatencyParams
	if params != nil {
		json.Unmarshal(params, &p)
	}
	if p.Count <= 0 {
		p.Count = 20
	}

	// Signal ready.
	if err := protocol.WriteMsg(conn, protocol.MsgTestReady, protocol.TestReady{}); err != nil {
		return nil, err
	}

	// Echo loop: read ping, send pong.
	buf := make([]byte, protocol.DataHeaderSize)
	for i := 0; i < p.Count; i++ {
		env, err := protocol.ReadMsg(conn)
		if err != nil {
			return nil, err
		}
		if env.Type != protocol.MsgTestStart {
			break
		}
		// Immediately echo back.
		if err := protocol.WriteMsg(conn, protocol.MsgTestStart, protocol.TestStart{}); err != nil {
			return nil, err
		}
	}
	_ = buf

	return nil, nil // Client computes its own metrics from RTTs.
}

// RunLatencyClient runs the latency test from the client side.
func RunLatencyClient(ctx context.Context, conn net.Conn, count int) (*LatencyMetrics, error) {
	if count <= 0 {
		count = 20
	}

	params, _ := json.Marshal(protocol.LatencyParams{Count: count})
	if err := protocol.WriteMsg(conn, protocol.MsgTestReq, protocol.TestRequest{
		Test:   protocol.TestLatency,
		Params: params,
	}); err != nil {
		return nil, err
	}

	// Wait for ready.
	env, err := protocol.ReadMsg(conn)
	if err != nil {
		return nil, err
	}
	if env.Type == protocol.MsgError {
		var e protocol.ErrorMsg
		protocol.DecodeBody(env, &e)
		return nil, &testError{e.Message}
	}

	// Ping-pong loop.
	rtts := make([]float64, 0, count)
	for i := 0; i < count; i++ {
		select {
		case <-ctx.Done():
			break
		default:
		}

		start := time.Now()
		if err := protocol.WriteMsg(conn, protocol.MsgTestStart, protocol.TestStart{}); err != nil {
			return nil, err
		}
		if _, err := protocol.ReadMsg(conn); err != nil {
			return nil, err
		}
		rtt := time.Since(start).Seconds() * 1000 // ms
		rtts = append(rtts, rtt)

		// Small delay between samples to avoid burst.
		time.Sleep(50 * time.Millisecond)
	}

	// Consume the server's test_result message.
	protocol.ReadMsg(conn)

	return computeLatencyMetrics(rtts), nil
}

func computeLatencyMetrics(samples []float64) *LatencyMetrics {
	if len(samples) == 0 {
		return &LatencyMetrics{}
	}

	sorted := make([]float64, len(samples))
	copy(sorted, samples)
	sort.Float64s(sorted)

	sum := 0.0
	for _, v := range sorted {
		sum += v
	}
	avg := sum / float64(len(sorted))

	variance := 0.0
	for _, v := range sorted {
		d := v - avg
		variance += d * d
	}
	variance /= float64(len(sorted))

	p95idx := int(float64(len(sorted)-1) * 0.95)

	return &LatencyMetrics{
		Samples: samples,
		Min:     sorted[0],
		Max:     sorted[len(sorted)-1],
		Avg:     avg,
		Median:  sorted[len(sorted)/2],
		P95:     sorted[p95idx],
		StdDev:  math.Sqrt(variance),
	}
}

type testError struct {
	msg string
}

func (e *testError) Error() string { return e.msg }

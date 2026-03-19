package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net"
	"time"

	"github.com/goozt/gospeed/internal/protocol"
	"github.com/goozt/gospeed/internal/server"
)

func init() {
	server.RegisterHandler(protocol.TestJitter, handleJitterServer)
}

func handleJitterServer(ctx context.Context, conn net.Conn, params json.RawMessage) (any, error) {
	var p protocol.JitterParams
	if params != nil {
		json.Unmarshal(params, &p)
	}
	if p.IntervalMs <= 0 {
		p.IntervalMs = 20
	}
	if p.Count <= 0 {
		p.Count = 200
	}

	udpAddr, err := net.ResolveUDPAddr("udp", ":0")
	if err != nil {
		return nil, err
	}
	udpConn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return nil, err
	}
	defer udpConn.Close()

	port := udpConn.LocalAddr().(*net.UDPAddr).Port
	if err := protocol.WriteMsg(conn, protocol.MsgTestReady, protocol.TestReady{DataPort: port}); err != nil {
		return nil, err
	}

	// Wait for start.
	env, err := protocol.ReadMsg(conn)
	if err != nil {
		return nil, err
	}
	if env.Type != protocol.MsgTestStart {
		return nil, fmt.Errorf("expected test_start, got %s", env.Type)
	}

	arrivals := receiveJitterPackets(udpConn, p)
	return computeJitterMetrics(arrivals, p.Count), nil
}

func receiveJitterPackets(udpConn *net.UDPConn, p protocol.JitterParams) []time.Time {
	timeout := time.Duration(p.Count*p.IntervalMs+5000) * time.Millisecond
	udpConn.SetReadDeadline(time.Now().Add(timeout))

	buf := make([]byte, 1500)
	arrivals := make([]time.Time, 0, p.Count)

	for i := 0; i < p.Count+10; i++ {
		n, _, err := udpConn.ReadFromUDP(buf)
		if err != nil {
			break
		}
		if n < protocol.DataHeaderSize {
			continue
		}
		arrivals = append(arrivals, time.Now())
	}
	return arrivals
}

func computeJitterMetrics(arrivals []time.Time, packetsSent int) JitterMetrics {
	if len(arrivals) < 2 {
		return JitterMetrics{PacketsRecv: len(arrivals), PacketsSent: packetsSent}
	}

	deltas := make([]float64, 0, len(arrivals)-1)
	for i := 1; i < len(arrivals); i++ {
		d := arrivals[i].Sub(arrivals[i-1]).Seconds() * 1000
		deltas = append(deltas, d)
	}

	jitters := make([]float64, 0, len(deltas)-1)
	for i := 1; i < len(deltas); i++ {
		jitters = append(jitters, math.Abs(deltas[i]-deltas[i-1]))
	}

	if len(jitters) == 0 {
		return JitterMetrics{PacketsRecv: len(arrivals), PacketsSent: packetsSent}
	}

	sum, minJ, maxJ := 0.0, jitters[0], jitters[0]
	for _, j := range jitters {
		sum += j
		if j < minJ {
			minJ = j
		}
		if j > maxJ {
			maxJ = j
		}
	}
	avg := sum / float64(len(jitters))

	variance := 0.0
	for _, j := range jitters {
		d := j - avg
		variance += d * d
	}
	variance /= float64(len(jitters))

	return JitterMetrics{
		AvgJitter:   avg,
		MaxJitter:   maxJ,
		MinJitter:   minJ,
		StdDev:      math.Sqrt(variance),
		PacketsRecv: len(arrivals),
		PacketsSent: packetsSent,
	}
}

// RunJitterClient runs the jitter test from the client side.
func RunJitterClient(ctx context.Context, conn net.Conn, serverAddr string, intervalMs, count int) (*JitterMetrics, error) {
	if intervalMs <= 0 {
		intervalMs = 20
	}
	if count <= 0 {
		count = 200
	}

	params, _ := json.Marshal(protocol.JitterParams{IntervalMs: intervalMs, Count: count})
	if err := protocol.WriteMsg(conn, protocol.MsgTestReq, protocol.TestRequest{
		Test:   protocol.TestJitter,
		Params: params,
	}); err != nil {
		return nil, err
	}

	env, err := protocol.ReadMsg(conn)
	if err != nil {
		return nil, err
	}
	if env.Type == protocol.MsgError {
		var e protocol.ErrorMsg
		protocol.DecodeBody(env, &e)
		return nil, &testError{e.Message}
	}
	var ready protocol.TestReady
	protocol.DecodeBody(env, &ready)

	host, _, _ := net.SplitHostPort(serverAddr)
	udpAddr := fmt.Sprintf("%s:%d", host, ready.DataPort)
	raddr, err := net.ResolveUDPAddr("udp", udpAddr)
	if err != nil {
		return nil, err
	}
	udpConn, err := net.DialUDP("udp", nil, raddr)
	if err != nil {
		return nil, err
	}
	defer udpConn.Close()

	// Signal start.
	if err := protocol.WriteMsg(conn, protocol.MsgTestStart, protocol.TestStart{}); err != nil {
		return nil, err
	}

	// Send packets at fixed intervals.
	interval := time.Duration(intervalMs) * time.Millisecond
	buf := make([]byte, 64) // small packets for jitter measurement
	for i := 0; i < count; i++ {
		select {
		case <-ctx.Done():
			goto done
		default:
		}
		EncodeDataHeader(buf, uint64(i))
		udpConn.Write(buf)
		time.Sleep(interval)
	}

done:
	// Read server result.
	env, err = protocol.ReadMsg(conn)
	if err != nil {
		return nil, err
	}
	var result JitterMetrics
	if env.Type == protocol.MsgTestResult {
		var msg protocol.TestResultMsg
		protocol.DecodeBody(env, &msg)
		json.Unmarshal(msg.Metrics, &result)
	}
	return &result, nil
}

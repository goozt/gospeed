package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"time"

	"github.com/goozt/gospeed/internal/protocol"
	"github.com/goozt/gospeed/internal/server"
)

func init() {
	server.RegisterHandler(protocol.TestUDP, handleUDPServer)
}

func handleUDPServer(ctx context.Context, conn net.Conn, params json.RawMessage) (any, error) {
	var p protocol.UDPParams
	if params != nil {
		json.Unmarshal(params, &p)
	}
	if p.Duration <= 0 {
		p.Duration = 10
	}
	if p.PacketSize <= 0 {
		p.PacketSize = 1400
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

	// Wait for start signal.
	env, err := protocol.ReadMsg(conn)
	if err != nil {
		return nil, err
	}
	if env.Type != protocol.MsgTestStart {
		return nil, fmt.Errorf("expected test_start, got %s", env.Type)
	}

	duration := time.Duration(p.Duration) * time.Second
	deadline := time.Now().Add(duration + 2*time.Second)
	udpConn.SetReadDeadline(deadline)

	buf := make([]byte, 10000)
	var packetsRecv int64
	var bytesTotal int64
	var outOfOrder int64
	var lastSeq uint64
	start := time.Now()

	for {
		n, _, err := udpConn.ReadFromUDP(buf)
		if err != nil {
			break
		}
		if n < protocol.DataHeaderSize {
			continue
		}
		hdr := DecodeDataHeader(buf[:protocol.DataHeaderSize])
		packetsRecv++
		bytesTotal += int64(n)

		if hdr.Seq < lastSeq {
			outOfOrder++
		}
		lastSeq = hdr.Seq
	}

	elapsed := time.Since(start).Seconds()

	return UDPMetrics{
		Duration:    elapsed,
		BytesTotal:  bytesTotal,
		BitsPerSec:  float64(bytesTotal) * 8 / elapsed,
		PacketsRecv: packetsRecv,
		OutOfOrder:  outOfOrder,
	}, nil
}

// RunUDPClient runs the UDP throughput + loss test from the client side.
func RunUDPClient(ctx context.Context, conn net.Conn, serverAddr string, duration int, packetSize int, bandwidth int64, progress func(float64)) (*UDPMetrics, error) {
	if duration <= 0 {
		duration = 10
	}
	if packetSize <= 0 {
		packetSize = 1400
	}

	params, _ := json.Marshal(protocol.UDPParams{
		Duration:   duration,
		PacketSize: packetSize,
		Bandwidth:  bandwidth,
	})
	if err := protocol.WriteMsg(conn, protocol.MsgTestReq, protocol.TestRequest{
		Test:   protocol.TestUDP,
		Params: params,
	}); err != nil {
		return nil, err
	}

	ready, err := readTestReady(conn)
	if err != nil {
		return nil, err
	}

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

	if err := protocol.WriteMsg(conn, protocol.MsgTestStart, protocol.TestStart{}); err != nil {
		return nil, err
	}

	sent, elapsed := udpSendLoop(ctx, udpConn, duration, packetSize, bandwidth)

	// Read server result and merge.
	serverResult := readUDPServerResult(conn)
	serverResult.PacketsSent = sent
	serverResult.PacketsLost = sent - serverResult.PacketsRecv
	if sent > 0 {
		serverResult.LossPercent = float64(serverResult.PacketsLost) / float64(sent) * 100
	}
	if serverResult.Duration == 0 {
		serverResult.Duration = elapsed
	}

	return &serverResult, nil
}

func udpSendLoop(ctx context.Context, udpConn *net.UDPConn, duration, packetSize int, bandwidth int64) (int64, float64) {
	dur := time.Duration(duration) * time.Second
	deadline := time.Now().Add(dur)
	start := time.Now()

	var delay time.Duration
	if bandwidth > 0 {
		packetsPerSec := float64(bandwidth) / 8 / float64(packetSize)
		delay = time.Duration(float64(time.Second) / packetsPerSec)
	}

	buf := make([]byte, packetSize)
	var seq uint64
	var sent int64
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return sent, time.Since(start).Seconds()
		default:
		}
		EncodeDataHeader(buf, seq)
		if _, err := udpConn.Write(buf); err != nil {
			break
		}
		seq++
		sent++
		if delay > 0 {
			time.Sleep(delay)
		}
	}
	return sent, time.Since(start).Seconds()
}

func readUDPServerResult(conn net.Conn) UDPMetrics {
	env, err := protocol.ReadMsg(conn)
	if err != nil {
		return UDPMetrics{}
	}
	var serverResult UDPMetrics
	if env.Type == protocol.MsgTestResult {
		var result protocol.TestResultMsg
		protocol.DecodeBody(env, &result)
		json.Unmarshal(result.Metrics, &serverResult)
	}
	return serverResult
}

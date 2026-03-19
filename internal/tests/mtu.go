package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"time"

	"github.com/goozt/gospeed/internal/platform"
	"github.com/goozt/gospeed/internal/protocol"
	"github.com/goozt/gospeed/internal/server"
)

func init() {
	server.RegisterHandler(protocol.TestMTU, handleMTUServer)
}

// MTU server: receive UDP packets of varying sizes and report the largest that arrived.
func handleMTUServer(ctx context.Context, conn net.Conn, params json.RawMessage) (any, error) {
	var p protocol.MTUParams
	if params != nil {
		json.Unmarshal(params, &p)
	}
	if p.MinSize <= 0 {
		p.MinSize = 576
	}
	if p.MaxSize <= 0 {
		p.MaxSize = 9000
	}

	// Open a UDP listener on an ephemeral port.
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

	// Tell client which port to send to.
	if err := protocol.WriteMsg(conn, protocol.MsgTestReady, protocol.TestReady{DataPort: port}); err != nil {
		return nil, err
	}

	// Receive loop: track the maximum size that arrives.
	// The client sends probes of various sizes and a 0xFF terminator.
	buf := make([]byte, 10000)
	maxReceived := 0
	for {
		select {
		case <-ctx.Done():
			return MTUMetrics{MTU: maxReceived}, nil
		default:
		}

		udpConn.SetReadDeadline(time.Now().Add(10 * time.Second))
		n, _, err := udpConn.ReadFromUDP(buf)
		if err != nil {
			break
		}
		if n == 0 {
			break
		}
		// Packet with only header byte 0xFF means "end of test".
		if n == 1 && buf[0] == 0xFF {
			break
		}
		if n > maxReceived {
			maxReceived = n
		}
	}

	return MTUMetrics{MTU: maxReceived + 28}, nil // +28 for IP(20)+UDP(8) headers
}

// RunMTUClient discovers the path MTU by sending UDP probes of varying sizes
// and letting the server report the largest one that arrived.
//
// The protocol is fire-and-forget: the client sends all probes without waiting
// for per-probe TCP confirmations, then reads a single TCP result from the
// server. This avoids blocking on the shared TCP control connection (which
// corrupts framing if deadlines are used and hangs if UDP is firewalled).
func RunMTUClient(ctx context.Context, conn net.Conn, serverAddr string) (*MTUMetrics, error) {
	params, _ := json.Marshal(protocol.MTUParams{MinSize: 576, MaxSize: 9000})
	if err := protocol.WriteMsg(conn, protocol.MsgTestReq, protocol.TestRequest{
		Test:   protocol.TestMTU,
		Params: params,
	}); err != nil {
		return nil, err
	}

	// Wait for ready with data port.
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

	// Resolve server host for UDP.
	host, _, err := net.SplitHostPort(serverAddr)
	if err != nil {
		host = serverAddr
	}
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

	// Set DF bit so oversized packets are dropped rather than fragmented.
	platform.SetDontFragment(udpConn)

	// Send probes in descending size order. Use binary search on Write errors
	// to find the local interface limit, then send probes for all sizes from
	// that limit down to min so the server can determine what actually arrives.
	minPayload, maxPayload := 548, 8972 // MTU - 28 for IP+UDP headers

	// First, find the local interface limit via binary search on Write errors.
	low, high := minPayload, maxPayload
	localMax := minPayload
	for low <= high {
		mid := (low + high) / 2
		payload := make([]byte, mid)
		_, err := udpConn.Write(payload)
		if err != nil {
			high = mid - 1
		} else {
			localMax = mid
			low = mid + 1
		}
	}

	// Now send probes from localMax down to minPayload so the server can
	// determine the path MTU (largest packet that actually arrives).
	// Use a step size to keep probe count reasonable (~20 probes).
	step := max((localMax-minPayload)/20, 1)
	for size := localMax; size >= minPayload; size -= step {
		select {
		case <-ctx.Done():
			return &MTUMetrics{MTU: minPayload + 28}, nil
		default:
		}
		payload := make([]byte, size)
		udpConn.Write(payload)
		// Small delay between probes to avoid burst loss.
		time.Sleep(10 * time.Millisecond)
	}
	// Always send the minimum size as a baseline.
	udpConn.Write(make([]byte, minPayload))
	time.Sleep(50 * time.Millisecond)

	// Signal end of test.
	udpConn.Write([]byte{0xFF})
	time.Sleep(100 * time.Millisecond)
	// Send terminator multiple times in case of packet loss.
	udpConn.Write([]byte{0xFF})

	// Read server's test result with a timeout. The server reports the largest
	// packet size it actually received. Use a goroutine so we don't block the
	// shared TCP connection indefinitely if the server is stuck.
	type readResult struct {
		env *protocol.Envelope
		err error
	}
	ch := make(chan readResult, 1)
	go func() {
		e, err := protocol.ReadMsg(conn)
		ch <- readResult{e, err}
	}()

	select {
	case res := <-ch:
		if res.err == nil && res.env.Type == protocol.MsgTestResult {
			var result protocol.TestResultMsg
			protocol.DecodeBody(res.env, &result)
			var serverMetrics MTUMetrics
			if json.Unmarshal(result.Metrics, &serverMetrics) == nil && serverMetrics.MTU > 0 {
				return &serverMetrics, nil
			}
		}
	case <-time.After(15 * time.Second):
		// Server didn't respond in time (likely UDP fully blocked or old server
		// with no read timeout). Fall back to local interface MTU. The blocked
		// goroutine will be cleaned up when runTest closes the connection.
	case <-ctx.Done():
	}

	return &MTUMetrics{MTU: localMax + 28}, nil
}

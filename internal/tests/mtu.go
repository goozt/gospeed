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

	udpConn, err := dialMTUProbe(serverAddr, ready.DataPort)
	if err != nil {
		return nil, err
	}
	defer udpConn.Close()

	localMax := sendMTUProbes(ctx, udpConn)

	return readMTUResult(ctx, conn, localMax)
}

func dialMTUProbe(serverAddr string, dataPort int) (*net.UDPConn, error) {
	host, _, err := net.SplitHostPort(serverAddr)
	if err != nil {
		host = serverAddr
	}
	raddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", host, dataPort))
	if err != nil {
		return nil, err
	}
	udpConn, err := net.DialUDP("udp", nil, raddr)
	if err != nil {
		return nil, err
	}
	platform.SetDontFragment(udpConn)
	return udpConn, nil
}

func sendMTUProbes(ctx context.Context, udpConn *net.UDPConn) int {
	minPayload, maxPayload := 548, 8972 // MTU - 28 for IP+UDP headers

	// Binary search for local interface limit.
	low, high := minPayload, maxPayload
	localMax := minPayload
	for low <= high {
		mid := (low + high) / 2
		if _, err := udpConn.Write(make([]byte, mid)); err != nil {
			high = mid - 1
		} else {
			localMax = mid
			low = mid + 1
		}
	}

	// Send probes from localMax down to minPayload.
	step := max((localMax-minPayload)/20, 1)
	for size := localMax; size >= minPayload; size -= step {
		select {
		case <-ctx.Done():
			return localMax
		default:
		}
		udpConn.Write(make([]byte, size))
		time.Sleep(10 * time.Millisecond)
	}
	udpConn.Write(make([]byte, minPayload))
	time.Sleep(50 * time.Millisecond)

	// Signal end of test (send twice for reliability).
	udpConn.Write([]byte{0xFF})
	time.Sleep(100 * time.Millisecond)
	udpConn.Write([]byte{0xFF})

	return localMax
}

func readMTUResult(ctx context.Context, conn net.Conn, localMax int) (*MTUMetrics, error) {
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
		// Server didn't respond; fall back to local interface MTU.
	case <-ctx.Done():
	}

	return &MTUMetrics{MTU: localMax + 28}, nil
}

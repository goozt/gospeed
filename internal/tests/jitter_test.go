package tests

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/goozt/gospeed/internal/protocol"
)

func TestJitterLoopback(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	serverDone := startJitterServer(t, ln)

	conn, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	// Handshake.
	protocol.WriteMsg(conn, protocol.MsgHello, protocol.Hello{Version: 1, ClientID: "test"})
	protocol.ReadMsg(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Use small count and short interval for fast test.
	metrics, err := RunJitterClient(ctx, conn, ln.Addr().String(), 10, 20)
	if err != nil {
		t.Fatalf("RunJitterClient: %v", err)
	}

	if metrics.PacketsSent <= 0 {
		t.Error("expected packets sent > 0")
	}
	if metrics.PacketsRecv <= 0 {
		t.Error("expected packets received > 0")
	}
	// On loopback, jitter should be very small.
	if metrics.AvgJitter < 0 {
		t.Error("avg jitter should be >= 0")
	}

	protocol.WriteMsg(conn, protocol.MsgGoodbye, protocol.Goodbye{})

	select {
	case err := <-serverDone:
		if err != nil {
			t.Errorf("server error: %v", err)
		}
	case <-time.After(15 * time.Second):
		t.Error("server didn't finish")
	}
}

func startJitterServer(t *testing.T, ln net.Listener) chan error {
	t.Helper()
	done := make(chan error, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			done <- err
			return
		}
		defer conn.Close()

		// Handshake.
		protocol.ReadMsg(conn)
		protocol.WriteMsg(conn, protocol.MsgHelloAck, protocol.HelloAck{Version: 1, SessionID: "s"})

		// Handle jitter test.
		env, _ := protocol.ReadMsg(conn)
		var req protocol.TestRequest
		protocol.DecodeBody(env, &req)

		result, err := handleJitterServer(context.Background(), conn, req.Params)
		if err != nil {
			done <- err
			return
		}

		metrics, _ := jsonMarshal(result)
		protocol.WriteMsg(conn, protocol.MsgTestResult, protocol.TestResultMsg{
			Test:    protocol.TestJitter,
			Metrics: metrics,
		})

		protocol.ReadMsg(conn) // goodbye
		done <- nil
	}()
	return done
}

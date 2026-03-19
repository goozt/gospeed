package tests

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/goozt/gospeed/internal/protocol"
)

func TestLatencyLoopback(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	serverDone := startLatencyServer(t, ln, 5)

	conn, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	protocol.WriteMsg(conn, protocol.MsgHello, protocol.Hello{Version: 1, ClientID: "test"})
	protocol.ReadMsg(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	metrics, err := RunLatencyClient(ctx, conn, 5)
	if err != nil {
		t.Fatalf("RunLatencyClient: %v", err)
	}

	if len(metrics.Samples) != 5 {
		t.Errorf("samples = %d, want 5", len(metrics.Samples))
	}
	if metrics.Min < 0 {
		t.Error("min should be >= 0")
	}
	if metrics.Min > metrics.Avg {
		t.Error("min should be <= avg")
	}
	if metrics.Avg > metrics.Max {
		t.Error("avg should be <= max")
	}

	protocol.WriteMsg(conn, protocol.MsgGoodbye, protocol.Goodbye{})

	select {
	case err := <-serverDone:
		if err != nil {
			t.Errorf("server error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Error("server didn't finish")
	}
}

func startLatencyServer(t *testing.T, ln net.Listener, count int) chan error {
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

		// Handle latency test request.
		protocol.ReadMsg(conn)
		protocol.WriteMsg(conn, protocol.MsgTestReady, protocol.TestReady{})

		for i := 0; i < count; i++ {
			if _, err := protocol.ReadMsg(conn); err != nil {
				break
			}
			protocol.WriteMsg(conn, protocol.MsgTestStart, protocol.TestStart{})
		}

		protocol.WriteMsg(conn, protocol.MsgTestResult, protocol.TestResultMsg{Test: protocol.TestLatency})
		protocol.ReadMsg(conn) // goodbye
		done <- nil
	}()
	return done
}

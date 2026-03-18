package tests

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/goozt/gospeed/internal/protocol"
)

func TestLatencyLoopback(t *testing.T) {
	// Start a minimal server that handles handshake + latency test.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	serverDone := make(chan error, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			serverDone <- err
			return
		}
		defer conn.Close()

		// Handshake.
		env, err := protocol.ReadMsg(conn)
		if err != nil {
			serverDone <- err
			return
		}
		if env.Type != protocol.MsgHello {
			serverDone <- err
			return
		}
		protocol.WriteMsg(conn, protocol.MsgHelloAck, protocol.HelloAck{
			Version:   1,
			SessionID: "test-session",
		})

		// Handle latency test request.
		env, err = protocol.ReadMsg(conn)
		if err != nil {
			serverDone <- err
			return
		}
		// Send ready.
		protocol.WriteMsg(conn, protocol.MsgTestReady, protocol.TestReady{})

		// Echo 5 pings.
		for i := 0; i < 5; i++ {
			env, err = protocol.ReadMsg(conn)
			if err != nil {
				break
			}
			protocol.WriteMsg(conn, protocol.MsgTestStart, protocol.TestStart{})
		}

		// Send test result.
		protocol.WriteMsg(conn, protocol.MsgTestResult, protocol.TestResultMsg{
			Test: protocol.TestLatency,
		})

		// Read goodbye.
		protocol.ReadMsg(conn)
		serverDone <- nil
	}()

	// Client side.
	conn, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	// Handshake.
	protocol.WriteMsg(conn, protocol.MsgHello, protocol.Hello{Version: 1, ClientID: "test"})
	protocol.ReadMsg(conn) // hello_ack

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	metrics, err := RunLatencyClient(ctx, conn, 5)
	if err != nil {
		t.Fatalf("RunLatencyClient: %v", err)
	}

	if len(metrics.Samples) != 5 {
		t.Errorf("samples = %d, want 5", len(metrics.Samples))
	}
	// On loopback, RTT may be 0 on some platforms due to timer resolution.
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

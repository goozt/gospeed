package tests

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/goozt/gospeed/internal/protocol"
)

func TestUDPLoopback(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	serverDone := startUDPServer(t, ln)

	conn, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	// Handshake.
	protocol.WriteMsg(conn, protocol.MsgHello, protocol.Hello{Version: 1, ClientID: "test"})
	protocol.ReadMsg(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	metrics, err := RunUDPClient(ctx, conn, ln.Addr().String(), 2, 1400, 0, nil)
	if err != nil {
		t.Fatalf("RunUDPClient: %v", err)
	}

	if metrics.PacketsSent <= 0 {
		t.Error("expected packets sent > 0")
	}
	if metrics.PacketsRecv <= 0 {
		t.Error("expected packets received > 0")
	}
	// On loopback, loss should be minimal.
	if metrics.LossPercent > 10 {
		t.Errorf("loss = %.2f%%, expected < 10%% on loopback", metrics.LossPercent)
	}
	if metrics.Duration <= 0 {
		t.Error("expected duration > 0")
	}

	protocol.WriteMsg(conn, protocol.MsgGoodbye, protocol.Goodbye{})

	select {
	case err := <-serverDone:
		if err != nil {
			t.Errorf("server error: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Error("server didn't finish")
	}
}

func startUDPServer(t *testing.T, ln net.Listener) chan error {
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

		// Handle UDP test.
		env, _ := protocol.ReadMsg(conn)
		var req protocol.TestRequest
		protocol.DecodeBody(env, &req)

		result, err := handleUDPServer(context.Background(), conn, req.Params)
		if err != nil {
			done <- err
			return
		}

		metrics, _ := jsonMarshal(result)
		protocol.WriteMsg(conn, protocol.MsgTestResult, protocol.TestResultMsg{
			Test:    protocol.TestUDP,
			Metrics: metrics,
		})

		protocol.ReadMsg(conn) // goodbye
		done <- nil
	}()
	return done
}

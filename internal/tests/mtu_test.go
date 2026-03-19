package tests

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/goozt/gospeed/internal/protocol"
)

func TestMTULoopback(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	serverDone := startMTUServer(t, ln)

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

	metrics, err := RunMTUClient(ctx, conn, ln.Addr().String())
	if err != nil {
		t.Fatalf("RunMTUClient: %v", err)
	}

	// On loopback, MTU should be at least 576 (minimum IP MTU).
	if metrics.MTU < 576 {
		t.Errorf("MTU = %d, expected >= 576", metrics.MTU)
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

func startMTUServer(t *testing.T, ln net.Listener) chan error {
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

		// Handle MTU test.
		env, _ := protocol.ReadMsg(conn)
		var req protocol.TestRequest
		protocol.DecodeBody(env, &req)

		result, err := handleMTUServer(context.Background(), conn, req.Params)
		if err != nil {
			done <- err
			return
		}

		metrics, _ := jsonMarshal(result)
		protocol.WriteMsg(conn, protocol.MsgTestResult, protocol.TestResultMsg{
			Test:    protocol.TestMTU,
			Metrics: metrics,
		})

		protocol.ReadMsg(conn) // goodbye
		done <- nil
	}()
	return done
}

package tests

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/goozt/gospeed/internal/protocol"
)

func TestTCPUploadLoopback(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	serverDone := startTCPServer(t, ln)

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

	metrics, err := RunTCPClient(ctx, conn, ln.Addr().String(), 2, 2, false, nil)
	if err != nil {
		t.Fatalf("RunTCPClient upload: %v", err)
	}

	if metrics.Direction != "upload" {
		t.Errorf("direction = %s, want upload", metrics.Direction)
	}
	if metrics.BytesTotal <= 0 {
		t.Error("expected bytes transferred > 0")
	}
	if metrics.BitsPerSec <= 0 {
		t.Error("expected throughput > 0")
	}
	if metrics.Streams != 2 {
		t.Errorf("streams = %d, want 2", metrics.Streams)
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

func TestTCPDownloadLoopback(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	serverDone := startTCPServer(t, ln)

	conn, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	protocol.WriteMsg(conn, protocol.MsgHello, protocol.Hello{Version: 1, ClientID: "test"})
	protocol.ReadMsg(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	metrics, err := RunTCPClient(ctx, conn, ln.Addr().String(), 2, 2, true, nil)
	if err != nil {
		t.Fatalf("RunTCPClient download: %v", err)
	}

	if metrics.Direction != "download" {
		t.Errorf("direction = %s, want download", metrics.Direction)
	}
	if metrics.BytesTotal <= 0 {
		t.Error("expected bytes transferred > 0")
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

// startTCPServer runs a minimal server handling handshake + TCP test.
func startTCPServer(t *testing.T, ln net.Listener) chan error {
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

		// Handle TCP test.
		env, _ := protocol.ReadMsg(conn) // test_req
		var req protocol.TestRequest
		protocol.DecodeBody(env, &req)

		result, err := handleTCPServer(context.Background(), conn, req.Params)
		if err != nil {
			done <- err
			return
		}

		metrics, _ := jsonMarshal(result)
		protocol.WriteMsg(conn, protocol.MsgTestResult, protocol.TestResultMsg{
			Test:    protocol.TestTCP,
			Metrics: metrics,
		})

		protocol.ReadMsg(conn) // goodbye
		done <- nil
	}()
	return done
}

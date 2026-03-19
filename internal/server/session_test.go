package server

import (
	"context"
	"encoding/json"
	"net"
	"testing"
	"time"

	"github.com/goozt/gospeed/internal/protocol"
)

func TestHandshakeSuccess(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	done := make(chan error, 1)
	go func() {
		ctx := context.Background()
		sess, err := newSession(ctx, server)
		if err != nil {
			done <- err
			return
		}
		if sess.ID == "" {
			t.Error("session ID should not be empty")
		}
		if sess.clientID != "test-client" {
			t.Errorf("clientID = %s, want test-client", sess.clientID)
		}
		done <- nil
	}()

	// Client sends hello.
	protocol.WriteMsg(client, protocol.MsgHello, protocol.Hello{
		Version:  protocol.ProtocolVersion,
		ClientID: "test-client",
	})

	// Read ack.
	env, err := protocol.ReadMsg(client)
	if err != nil {
		t.Fatalf("read ack: %v", err)
	}
	if env.Type != protocol.MsgHelloAck {
		t.Fatalf("expected hello_ack, got %s", env.Type)
	}

	var ack protocol.HelloAck
	protocol.DecodeBody(env, &ack)
	if ack.Version != protocol.ProtocolVersion {
		t.Errorf("ack version = %d, want %d", ack.Version, protocol.ProtocolVersion)
	}
	if ack.SessionID == "" {
		t.Error("ack session_id should not be empty")
	}

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("session error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout")
	}
}

func TestHandshakeWrongVersion(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	done := make(chan error, 1)
	go func() {
		_, err := newSession(context.Background(), server)
		done <- err
	}()

	// Send hello with wrong version.
	protocol.WriteMsg(client, protocol.MsgHello, protocol.Hello{
		Version:  999,
		ClientID: "test",
	})

	err := <-done
	if err == nil {
		t.Fatal("expected error for wrong protocol version")
	}
}

func TestHandshakeWrongMessageType(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	done := make(chan error, 1)
	go func() {
		_, err := newSession(context.Background(), server)
		done <- err
	}()

	// Send goodbye instead of hello.
	protocol.WriteMsg(client, protocol.MsgGoodbye, protocol.Goodbye{})

	err := <-done
	if err == nil {
		t.Fatal("expected error for wrong message type")
	}
}

func TestSessionRunGoodbye(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()

	done := make(chan error, 1)
	go func() {
		sess, err := newSession(context.Background(), serverConn)
		if err != nil {
			done <- err
			return
		}
		done <- sess.Run()
	}()

	// Client: handshake then goodbye.
	protocol.WriteMsg(clientConn, protocol.MsgHello, protocol.Hello{
		Version: protocol.ProtocolVersion, ClientID: "test",
	})
	protocol.ReadMsg(clientConn) // ack

	protocol.WriteMsg(clientConn, protocol.MsgGoodbye, protocol.Goodbye{})

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("session.Run: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout")
	}
}

func TestSessionRunUnknownTest(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()

	done := make(chan error, 1)
	go func() {
		sess, err := newSession(context.Background(), serverConn)
		if err != nil {
			done <- err
			return
		}
		done <- sess.Run()
	}()

	// Handshake.
	protocol.WriteMsg(clientConn, protocol.MsgHello, protocol.Hello{
		Version: protocol.ProtocolVersion, ClientID: "test",
	})
	protocol.ReadMsg(clientConn) // ack

	// Request unknown test.
	params, _ := json.Marshal(nil)
	protocol.WriteMsg(clientConn, protocol.MsgTestReq, protocol.TestRequest{
		Test:   "nonexistent",
		Params: params,
	})

	// Should get an error message back.
	env, err := protocol.ReadMsg(clientConn)
	if err != nil {
		t.Fatalf("read error msg: %v", err)
	}
	if env.Type != protocol.MsgError {
		t.Errorf("expected error, got %s", env.Type)
	}

	// Session should still be alive — send goodbye.
	protocol.WriteMsg(clientConn, protocol.MsgGoodbye, protocol.Goodbye{})

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("session.Run: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout")
	}
}

func TestSessionRunUnexpectedMessage(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()

	done := make(chan error, 1)
	go func() {
		sess, err := newSession(context.Background(), serverConn)
		if err != nil {
			done <- err
			return
		}
		done <- sess.Run()
	}()

	// Handshake.
	protocol.WriteMsg(clientConn, protocol.MsgHello, protocol.Hello{
		Version: protocol.ProtocolVersion, ClientID: "test",
	})
	protocol.ReadMsg(clientConn)

	// Send unexpected message type (hello again).
	protocol.WriteMsg(clientConn, protocol.MsgHello, protocol.Hello{Version: 1})

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected error for unexpected message type")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout")
	}
}

func TestSessionContextCancellation(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		sess, err := newSession(ctx, serverConn)
		if err != nil {
			done <- err
			return
		}
		done <- sess.Run()
	}()

	// Handshake.
	protocol.WriteMsg(clientConn, protocol.MsgHello, protocol.Hello{
		Version: protocol.ProtocolVersion, ClientID: "test",
	})
	protocol.ReadMsg(clientConn)

	// Cancel context and close the client conn to unblock ReadMsg.
	cancel()
	clientConn.Close()

	select {
	case err := <-done:
		_ = err // context error or read error, both acceptable
	case <-time.After(5 * time.Second):
		t.Fatal("timeout — session didn't respond to context cancellation")
	}
}

func TestRegisteredTests(t *testing.T) {
	// Register a dummy handler.
	RegisterHandler("test_dummy", func(ctx context.Context, conn net.Conn, params json.RawMessage) (any, error) {
		return nil, nil
	})
	defer func() {
		delete(handlers, "test_dummy")
	}()

	tests := RegisteredTests()
	found := false
	for _, tt := range tests {
		if tt == "test_dummy" {
			found = true
		}
	}
	if !found {
		t.Error("expected test_dummy in registered tests")
	}
}

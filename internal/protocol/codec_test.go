package protocol

import (
	"encoding/binary"
	"net"
	"strings"
	"testing"
)

func TestWriteReadMsg(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	hello := Hello{Version: 1, ClientID: "test-123"}

	go func() {
		if err := WriteMsg(client, MsgHello, hello); err != nil {
			t.Errorf("WriteMsg: %v", err)
		}
	}()

	env, err := ReadMsg(server)
	if err != nil {
		t.Fatalf("ReadMsg: %v", err)
	}
	if env.Type != MsgHello {
		t.Errorf("type = %s, want %s", env.Type, MsgHello)
	}

	var got Hello
	if err := DecodeBody(env, &got); err != nil {
		t.Fatalf("DecodeBody: %v", err)
	}
	if got.Version != 1 {
		t.Errorf("version = %d, want 1", got.Version)
	}
	if got.ClientID != "test-123" {
		t.Errorf("client_id = %s, want test-123", got.ClientID)
	}
}

func TestWriteReadMultipleMessages(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	go func() {
		WriteMsg(client, MsgHello, Hello{Version: 1, ClientID: "a"})
		WriteMsg(client, MsgHelloAck, HelloAck{Version: 1, SessionID: "s1"})
		WriteMsg(client, MsgGoodbye, Goodbye{})
	}()

	for _, want := range []MsgType{MsgHello, MsgHelloAck, MsgGoodbye} {
		env, err := ReadMsg(server)
		if err != nil {
			t.Fatalf("ReadMsg: %v", err)
		}
		if env.Type != want {
			t.Errorf("type = %s, want %s", env.Type, want)
		}
	}
}

func TestReadMsgClosedConn(t *testing.T) {
	server, client := net.Pipe()
	client.Close()

	_, err := ReadMsg(server)
	if err == nil {
		t.Error("expected error from closed conn")
	}
	server.Close()
}

func TestReadMsgTooLarge(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	// Write a header claiming a message larger than maxMessageSize.
	go func() {
		var hdr [4]byte
		binary.BigEndian.PutUint32(hdr[:], uint32(maxMessageSize+1))
		client.Write(hdr[:])
	}()

	_, err := ReadMsg(server)
	if err == nil {
		t.Fatal("expected error for oversized message")
	}
	if !strings.Contains(err.Error(), "too large") {
		t.Errorf("error = %v, want 'too large'", err)
	}
}

func TestWriteMsgTooLarge(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	// Create a payload that exceeds maxMessageSize when marshalled.
	bigStr := strings.Repeat("x", maxMessageSize)
	err := WriteMsg(client, MsgHello, Hello{ClientID: bigStr})
	if err == nil {
		t.Fatal("expected error for oversized message")
	}
	if !strings.Contains(err.Error(), "too large") {
		t.Errorf("error = %v, want 'too large'", err)
	}
}

func TestSessionIDWriteRead(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	var id [16]byte
	for i := range id {
		id[i] = byte(i)
	}

	go func() {
		WriteSessionID(client, id)
	}()

	got, err := ReadSessionID(server)
	if err != nil {
		t.Fatalf("ReadSessionID: %v", err)
	}
	if got != id {
		t.Errorf("session ID mismatch: got %v, want %v", got, id)
	}
}

func TestDecodeBodyInvalid(t *testing.T) {
	env := &Envelope{Type: MsgHello, Body: []byte(`{invalid`)}
	var hello Hello
	if err := DecodeBody(env, &hello); err == nil {
		t.Error("expected error for invalid JSON body")
	}
}

func TestAllMessageTypes(t *testing.T) {
	// Verify all message types can round-trip through WriteMsg/ReadMsg.
	tests := []struct {
		msgType MsgType
		body    any
	}{
		{MsgHello, Hello{Version: 1, ClientID: "c"}},
		{MsgHelloAck, HelloAck{Version: 1, SessionID: "s", Tests: []TestType{TestTCP, TestUDP}}},
		{MsgTestReq, TestRequest{Test: TestTCP}},
		{MsgTestReady, TestReady{DataPort: 12345}},
		{MsgTestStart, TestStart{}},
		{MsgTestResult, TestResultMsg{Test: TestLatency, Metrics: []byte(`{"min_ms":1}`)}},
		{MsgError, ErrorMsg{Code: 500, Message: "fail"}},
		{MsgGoodbye, Goodbye{}},
	}

	for _, tt := range tests {
		t.Run(string(tt.msgType), func(t *testing.T) {
			server, client := net.Pipe()
			defer server.Close()
			defer client.Close()

			go func() {
				WriteMsg(client, tt.msgType, tt.body)
			}()

			env, err := ReadMsg(server)
			if err != nil {
				t.Fatalf("ReadMsg: %v", err)
			}
			if env.Type != tt.msgType {
				t.Errorf("type = %s, want %s", env.Type, tt.msgType)
			}
		})
	}
}

func TestReadMsgPartialHeader(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()

	// Write only 2 bytes of header then close.
	go func() {
		client.Write([]byte{0x00, 0x01})
		client.Close()
	}()

	_, err := ReadMsg(server)
	if err == nil {
		t.Error("expected error for partial header")
	}
}

func TestReadMsgPartialPayload(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()

	// Write header claiming 100 bytes, then only 5 bytes, then close.
	go func() {
		var hdr [4]byte
		binary.BigEndian.PutUint32(hdr[:], 100)
		client.Write(hdr[:])
		client.Write([]byte("hello"))
		client.Close()
	}()

	_, err := ReadMsg(server)
	if err == nil {
		t.Error("expected error for partial payload")
	}
}

func TestReadMsgInvalidJSON(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	// Write valid header + invalid JSON payload.
	go func() {
		payload := []byte(`not json at all`)
		var hdr [4]byte
		binary.BigEndian.PutUint32(hdr[:], uint32(len(payload)))
		client.Write(hdr[:])
		client.Write(payload)
	}()

	_, err := ReadMsg(server)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "unmarshal") {
		t.Errorf("error = %v, want unmarshal error", err)
	}
}

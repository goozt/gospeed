package protocol

import (
	"net"
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

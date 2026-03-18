package server

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/goozt/gospeed/internal/protocol"
)

func TestConcurrentClients(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()

	srv := New(addr)
	go srv.ListenAndServeWithListener(ctx, ln)

	// Connect 5 clients concurrently.
	const numClients = 5
	var wg sync.WaitGroup

	for i := 0; i < numClients; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
			if err != nil {
				t.Errorf("client %d dial: %v", id, err)
				return
			}
			defer conn.Close()

			// Handshake.
			if err := protocol.WriteMsg(conn, protocol.MsgHello, protocol.Hello{
				Version:  1,
				ClientID: "client-" + string(rune('0'+id)),
			}); err != nil {
				t.Errorf("client %d hello: %v", id, err)
				return
			}

			env, err := protocol.ReadMsg(conn)
			if err != nil {
				t.Errorf("client %d read ack: %v", id, err)
				return
			}
			if env.Type != protocol.MsgHelloAck {
				t.Errorf("client %d: expected hello_ack, got %s", id, env.Type)
				return
			}

			// Goodbye.
			protocol.WriteMsg(conn, protocol.MsgGoodbye, protocol.Goodbye{})
		}(i)
	}

	wg.Wait()
}

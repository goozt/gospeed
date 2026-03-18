package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"time"

	"github.com/goozt/gospeed/internal/protocol"
)

// Session represents a single client connection.
type Session struct {
	ID       string
	conn     net.Conn
	ctx      context.Context
	cancel   context.CancelFunc
	clientID string
}

func newSession(ctx context.Context, conn net.Conn) (*Session, error) {
	ctx, cancel := context.WithCancel(ctx)
	s := &Session{
		conn:   conn,
		ctx:    ctx,
		cancel: cancel,
	}

	if err := s.handshake(); err != nil {
		cancel()
		return nil, fmt.Errorf("handshake: %w", err)
	}
	return s, nil
}

func (s *Session) handshake() error {
	s.conn.SetDeadline(time.Now().Add(10 * time.Second))
	defer s.conn.SetDeadline(time.Time{})

	env, err := protocol.ReadMsg(s.conn)
	if err != nil {
		return fmt.Errorf("read hello: %w", err)
	}
	if env.Type != protocol.MsgHello {
		return fmt.Errorf("expected hello, got %s", env.Type)
	}

	var hello protocol.Hello
	if err := protocol.DecodeBody(env, &hello); err != nil {
		return fmt.Errorf("decode hello: %w", err)
	}
	if hello.Version != protocol.ProtocolVersion {
		return fmt.Errorf("unsupported protocol version %d", hello.Version)
	}

	s.clientID = hello.ClientID

	// Generate session ID.
	var idBytes [8]byte
	rand.Read(idBytes[:])
	s.ID = hex.EncodeToString(idBytes[:])

	ack := protocol.HelloAck{
		Version:   protocol.ProtocolVersion,
		SessionID: s.ID,
	}
	return protocol.WriteMsg(s.conn, protocol.MsgHelloAck, ack)
}

// Run processes test requests until the client disconnects or context is cancelled.
func (s *Session) Run() error {
	defer s.cancel()

	for {
		select {
		case <-s.ctx.Done():
			return s.ctx.Err()
		default:
		}

		env, err := protocol.ReadMsg(s.conn)
		if err != nil {
			return fmt.Errorf("read message: %w", err)
		}

		switch env.Type {
		case protocol.MsgGoodbye:
			return nil
		case protocol.MsgTestReq:
			if err := s.handleTestRequest(env); err != nil {
				// Send error to client, but don't kill the session.
				protocol.WriteMsg(s.conn, protocol.MsgError, protocol.ErrorMsg{
					Code:    500,
					Message: err.Error(),
				})
			}
		default:
			return fmt.Errorf("unexpected message type: %s", env.Type)
		}
	}
}

func (s *Session) handleTestRequest(env *protocol.Envelope) (retErr error) {
	defer func() {
		if r := recover(); r != nil {
			retErr = fmt.Errorf("test panic: %v", r)
		}
	}()

	var req protocol.TestRequest
	if err := protocol.DecodeBody(env, &req); err != nil {
		return fmt.Errorf("decode test request: %w", err)
	}

	handler, ok := handlers[req.Test]
	if !ok {
		return fmt.Errorf("unknown test type: %s", req.Test)
	}

	result, err := handler(s.ctx, s.conn, req.Params)
	if err != nil {
		return fmt.Errorf("test %s: %w", req.Test, err)
	}

	metricsJSON, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("marshal result: %w", err)
	}

	return protocol.WriteMsg(s.conn, protocol.MsgTestResult, protocol.TestResultMsg{
		Test:    req.Test,
		Metrics: metricsJSON,
	})
}

// Conn returns the session's control connection (used by test handlers).
func (s *Session) Conn() net.Conn {
	return s.conn
}

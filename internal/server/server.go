package server

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
)

// Server accepts and manages concurrent client connections.
type Server struct {
	Addr     string
	listener net.Listener

	sessions sync.Map // sessionID string → *Session
	mu       sync.Mutex
	wg       sync.WaitGroup
}

// New creates a server bound to the given address.
func New(addr string) *Server {
	return &Server{Addr: addr}
}

// ListenAndServe starts accepting connections. Blocks until ctx is cancelled.
func (s *Server) ListenAndServe(ctx context.Context) error {
	ln, err := net.Listen("tcp", s.Addr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	s.listener = ln
	log.Printf("gospeed server listening on %s", s.Addr)

	go func() {
		<-ctx.Done()
		ln.Close()
	}()

	for {
		conn, err := ln.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				s.wg.Wait()
				return nil
			default:
				log.Printf("accept error: %v", err)
				continue
			}
		}
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			s.handleConn(ctx, conn)
		}()
	}
}

// ListenAndServeWithListener starts serving on an existing listener (for testing).
func (s *Server) ListenAndServeWithListener(ctx context.Context, ln net.Listener) error {
	s.listener = ln
	s.Addr = ln.Addr().String()

	go func() {
		<-ctx.Done()
		ln.Close()
	}()

	for {
		conn, err := ln.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				s.wg.Wait()
				return nil
			default:
				return fmt.Errorf("accept: %w", err)
			}
		}
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			s.handleConn(ctx, conn)
		}()
	}
}

func (s *Server) handleConn(ctx context.Context, conn net.Conn) {
	defer conn.Close()
	defer func() {
		if r := recover(); r != nil {
			log.Printf("panic handling connection from %s: %v", conn.RemoteAddr(), r)
		}
	}()

	sess, err := newSession(ctx, conn)
	if err != nil {
		log.Printf("session handshake failed: %v", err)
		return
	}

	s.sessions.Store(sess.ID, sess)
	defer s.sessions.Delete(sess.ID)

	log.Printf("session %s started from %s", sess.ID, conn.RemoteAddr())
	if err := sess.Run(); err != nil {
		log.Printf("session %s error: %v", sess.ID, err)
	}
	log.Printf("session %s ended", sess.ID)
}

// Listener returns the underlying net.Listener (for tests).
func (s *Server) Listener() net.Listener {
	return s.listener
}

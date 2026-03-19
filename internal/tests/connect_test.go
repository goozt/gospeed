package tests

import (
	"context"
	"net"
	"testing"
	"time"
)

func TestConnectClient(t *testing.T) {
	// Start a TCP listener to connect to.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	// Accept connections in background.
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			conn.Close()
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	metrics, err := RunConnectClient(ctx, ln.Addr().String(), 5)
	if err != nil {
		t.Fatalf("RunConnectClient: %v", err)
	}

	if len(metrics.Samples) != 5 {
		t.Errorf("samples = %d, want 5", len(metrics.Samples))
	}
	if metrics.Min < 0 {
		t.Error("min should be >= 0")
	}
	if metrics.Min > metrics.Avg {
		t.Error("min should be <= avg")
	}
	if metrics.Avg > metrics.Max {
		t.Error("avg should be <= max")
	}
	// On loopback, connect time should be very fast.
	if metrics.Max > 1000 {
		t.Errorf("max connect time = %.2f ms, expected < 1000 ms on loopback", metrics.Max)
	}
}

func TestConnectClientNoServer(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Use a port that's unlikely to be open.
	metrics, err := RunConnectClient(ctx, "127.0.0.1:1", 2)
	if err != nil {
		t.Fatalf("RunConnectClient: %v", err)
	}
	// All connections should fail, so no samples.
	if len(metrics.Samples) != 0 {
		t.Errorf("expected 0 samples with no server, got %d", len(metrics.Samples))
	}
}

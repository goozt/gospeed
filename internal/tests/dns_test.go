package tests

import (
	"context"
	"testing"
	"time"
)

func TestDNSClient(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	metrics, err := RunDNSClient(ctx, "localhost", 3)
	if err != nil {
		t.Fatalf("RunDNSClient: %v", err)
	}

	if metrics.Host != "localhost" {
		t.Errorf("host = %s, want localhost", metrics.Host)
	}
	if len(metrics.Samples) == 0 {
		t.Error("expected at least one sample")
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
}

func TestDNSClientCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	metrics, err := RunDNSClient(ctx, "localhost", 10)
	if err != nil {
		t.Fatalf("RunDNSClient: %v", err)
	}
	// Should return with zero or few samples due to cancellation.
	if len(metrics.Samples) > 1 {
		t.Errorf("expected <= 1 samples with cancelled ctx, got %d", len(metrics.Samples))
	}
}

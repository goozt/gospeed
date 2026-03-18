package tests

import (
	"context"
	"net"
	"time"
)

// RunDNSClient measures DNS resolution latency (client-side only, no server involvement).
func RunDNSClient(ctx context.Context, host string, count int) (*DNSMetrics, error) {
	if count <= 0 {
		count = 10
	}

	resolver := &net.Resolver{}
	samples := make([]float64, 0, count)

	for i := 0; i < count; i++ {
		if ctx.Err() != nil {
			break
		}

		start := time.Now()
		_, err := resolver.LookupHost(ctx, host)
		elapsed := time.Since(start).Seconds() * 1000 // ms
		if err != nil {
			continue // skip failed lookups
		}
		samples = append(samples, elapsed)
		time.Sleep(100 * time.Millisecond) // avoid burst
	}

	if len(samples) == 0 {
		return &DNSMetrics{Host: host}, nil
	}

	min, max, sum := samples[0], samples[0], 0.0
	for _, s := range samples {
		sum += s
		if s < min {
			min = s
		}
		if s > max {
			max = s
		}
	}

	return &DNSMetrics{
		Host:    host,
		Samples: samples,
		Min:     min,
		Max:     max,
		Avg:     sum / float64(len(samples)),
	}, nil
}

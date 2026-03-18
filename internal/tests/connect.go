package tests

import (
	"context"
	"net"
	"time"
)

// RunConnectClient measures TCP connection setup time (client-side only).
func RunConnectClient(ctx context.Context, serverAddr string, count int) (*ConnectMetrics, error) {
	if count <= 0 {
		count = 10
	}

	samples := make([]float64, 0, count)
	for i := 0; i < count; i++ {
		select {
		case <-ctx.Done():
			break
		default:
		}

		start := time.Now()
		conn, err := net.DialTimeout("tcp", serverAddr, 5*time.Second)
		elapsed := time.Since(start).Seconds() * 1000 // ms
		if err != nil {
			continue
		}
		conn.Close()
		samples = append(samples, elapsed)
		time.Sleep(100 * time.Millisecond)
	}

	if len(samples) == 0 {
		return &ConnectMetrics{}, nil
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

	return &ConnectMetrics{
		Samples: samples,
		Min:     min,
		Max:     max,
		Avg:     sum / float64(len(samples)),
	}, nil
}

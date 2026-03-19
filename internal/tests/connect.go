package tests

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net"
	"time"

	"github.com/goozt/gospeed/internal/protocol"
)

// RunConnectClient measures TCP connection setup time (client-side only).
func RunConnectClient(ctx context.Context, serverAddr string, count int) (*ConnectMetrics, error) {
	if count <= 0 {
		count = 10
	}

	samples := make([]float64, 0, count)
	for i := 0; i < count; i++ {
		if ctx.Err() != nil {
			break
		}

		start := time.Now()
		conn, err := net.DialTimeout("tcp", serverAddr, 5*time.Second)
		elapsed := time.Since(start).Seconds() * 1000 // ms
		if err != nil {
			continue
		}
		// Send a proper hello+goodbye so the server doesn't log handshake errors.
		go func(c net.Conn) {
			defer c.Close()
			var idBytes [8]byte
			rand.Read(idBytes[:])
			clientID := hex.EncodeToString(idBytes[:])
			if err := protocol.WriteMsg(c, protocol.MsgHello, protocol.Hello{
				Version:  protocol.ProtocolVersion,
				ClientID: clientID,
			}); err != nil {
				return
			}
			if _, err := protocol.ReadMsg(c); err != nil {
				return
			}
			protocol.WriteMsg(c, protocol.MsgGoodbye, protocol.Goodbye{})
		}(conn)
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

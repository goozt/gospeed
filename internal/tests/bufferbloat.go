package tests

import (
	"context"
	"encoding/json"
	"net"
	"time"

	"github.com/goozt/gospeed/internal/protocol"
	"github.com/goozt/gospeed/internal/server"
)

func init() {
	server.RegisterHandler(protocol.TestBufferbloat, handleBufferbloatServer)
}

// Server side: handle both the latency probes and the TCP load.
// The bufferbloat test reuses latency and TCP handlers internally,
// so the server just needs to handle the combined protocol.
func handleBufferbloatServer(ctx context.Context, conn net.Conn, params json.RawMessage) (any, error) {
	var p protocol.BufferbloatParams
	if params != nil {
		json.Unmarshal(params, &p)
	}
	if p.Duration <= 0 {
		p.Duration = 10
	}
	if p.Streams <= 0 {
		p.Streams = 4
	}

	// Open a TCP listener for throughput streams.
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		return nil, err
	}
	defer ln.Close()

	port := ln.Addr().(*net.TCPAddr).Port
	if err := protocol.WriteMsg(conn, protocol.MsgTestReady, protocol.TestReady{DataPort: port}); err != nil {
		return nil, err
	}

	// Wait for start.
	env, err := protocol.ReadMsg(conn)
	if err != nil {
		return nil, err
	}
	if env.Type != protocol.MsgTestStart {
		return nil, nil
	}

	duration := time.Duration(p.Duration) * time.Second
	deadline := time.Now().Add(duration)

	// Accept data connections for throughput load.
	go func() {
		for {
			dc, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				buf := make([]byte, 128*1024)
				for {
					c.SetReadDeadline(deadline.Add(2 * time.Second))
					_, err := c.Read(buf)
					if err != nil {
						return
					}
				}
			}(dc)
		}
	}()

	// Handle latency probes on control connection during the test.
	for time.Now().Before(deadline) {
		conn.SetReadDeadline(deadline.Add(time.Second))
		env, err := protocol.ReadMsg(conn)
		if err != nil {
			break
		}
		if env.Type == protocol.MsgTestStart {
			// Echo back immediately for latency measurement.
			protocol.WriteMsg(conn, protocol.MsgTestStart, protocol.TestStart{})
		} else if env.Type == protocol.MsgGoodbye {
			break
		}
	}
	conn.SetReadDeadline(time.Time{})

	return nil, nil // Client computes its own metrics.
}

// RunBufferbloatClient runs the bufferbloat/responsiveness test from the client side.
func RunBufferbloatClient(ctx context.Context, conn net.Conn, serverAddr string, duration, streams int, progress func(string)) (*BufferbloatMetrics, error) {
	if duration <= 0 {
		duration = 10
	}
	if streams <= 0 {
		streams = 4
	}

	// Step 1: Run unloaded latency first (reuse latency test separately before this).
	// We assume the caller has already run latency and passes baseline.
	// For standalone use, we measure baseline here.
	if progress != nil {
		progress("measuring baseline latency...")
	}
	baselineLatency, err := RunLatencyClient(ctx, conn, 10)
	if err != nil {
		return nil, err
	}

	// Step 2: Request bufferbloat test.
	params, _ := json.Marshal(protocol.BufferbloatParams{Duration: duration, Streams: streams})
	if err := protocol.WriteMsg(conn, protocol.MsgTestReq, protocol.TestRequest{
		Test:   protocol.TestBufferbloat,
		Params: params,
	}); err != nil {
		return nil, err
	}

	env, err := protocol.ReadMsg(conn)
	if err != nil {
		return nil, err
	}
	if env.Type == protocol.MsgError {
		var e protocol.ErrorMsg
		protocol.DecodeBody(env, &e)
		return nil, &testError{e.Message}
	}
	var ready protocol.TestReady
	protocol.DecodeBody(env, &ready)

	host, _, _ := net.SplitHostPort(serverAddr)

	// Signal start.
	if err := protocol.WriteMsg(conn, protocol.MsgTestStart, protocol.TestStart{}); err != nil {
		return nil, err
	}

	if progress != nil {
		progress("generating load and measuring latency...")
	}

	dur := time.Duration(duration) * time.Second
	deadline := time.Now().Add(dur)
	start := time.Now()

	// Start TCP load streams.
	dataAddr := net.JoinHostPort(host, itoa(ready.DataPort))
	loadDone := make(chan struct{})
	var totalBytes int64
	go func() {
		defer close(loadDone)
		conns := make([]net.Conn, 0, streams)
		for i := 0; i < streams; i++ {
			dc, err := net.Dial("tcp", dataAddr)
			if err != nil {
				continue
			}
			conns = append(conns, dc)
		}
		defer func() {
			for _, c := range conns {
				c.Close()
			}
		}()

		buf := make([]byte, 128*1024)
		for time.Now().Before(deadline) {
			for _, c := range conns {
				c.SetWriteDeadline(deadline)
				n, err := c.Write(buf)
				if err != nil {
					continue
				}
				totalBytes += int64(n)
			}
		}
	}()

	// Measure latency under load.
	loadedSamples := make([]float64, 0, 20)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	// Wait a second for load to ramp up.
	time.Sleep(time.Second)

	for {
		select {
		case <-ticker.C:
			if time.Now().After(deadline) {
				goto done
			}
			pingStart := time.Now()
			if err := protocol.WriteMsg(conn, protocol.MsgTestStart, protocol.TestStart{}); err != nil {
				goto done
			}
			if _, err := protocol.ReadMsg(conn); err != nil {
				goto done
			}
			rtt := time.Since(pingStart).Seconds() * 1000
			loadedSamples = append(loadedSamples, rtt)
		case <-ctx.Done():
			goto done
		}
	}

done:
	<-loadDone
	elapsed := time.Since(start).Seconds()

	// Signal end of latency probes.
	protocol.WriteMsg(conn, protocol.MsgGoodbye, protocol.Goodbye{})

	// Read server result.
	protocol.ReadMsg(conn)

	loadedLatency := computeLatencyMetrics(loadedSamples)
	rpm := 0.0
	if loadedLatency.Avg > 0 {
		rpm = 60000 / loadedLatency.Avg
	}

	return &BufferbloatMetrics{
		UnloadedLatency: *baselineLatency,
		LoadedLatency:   *loadedLatency,
		RPM:             rpm,
		LatencyIncrease: loadedLatency.Avg - baselineLatency.Avg,
		Throughput: TCPMetrics{
			Direction:  "upload",
			Duration:   elapsed,
			BytesTotal: totalBytes,
			BitsPerSec: float64(totalBytes) * 8 / elapsed,
			Streams:    streams,
		},
	}, nil
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	buf := make([]byte, 0, 10)
	for i > 0 {
		buf = append(buf, byte('0'+i%10))
		i /= 10
	}
	// Reverse.
	for l, r := 0, len(buf)-1; l < r; l, r = l+1, r-1 {
		buf[l], buf[r] = buf[r], buf[l]
	}
	return string(buf)
}

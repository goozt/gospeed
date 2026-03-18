package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/goozt/gospeed/internal/platform"
	"github.com/goozt/gospeed/internal/protocol"
	"github.com/goozt/gospeed/internal/server"
)

func init() {
	server.RegisterHandler(protocol.TestTCP, handleTCPServer)
}

func handleTCPServer(ctx context.Context, conn net.Conn, params json.RawMessage) (any, error) {
	var p protocol.TCPParams
	if params != nil {
		json.Unmarshal(params, &p)
	}
	if p.Duration <= 0 {
		p.Duration = 10
	}
	if p.Streams <= 0 {
		p.Streams = 4
	}

	// Open a TCP listener for data streams.
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		return nil, err
	}
	defer ln.Close()

	port := ln.Addr().(*net.TCPAddr).Port
	if err := protocol.WriteMsg(conn, protocol.MsgTestReady, protocol.TestReady{DataPort: port}); err != nil {
		return nil, err
	}

	// Wait for test_start signal.
	env, err := protocol.ReadMsg(conn)
	if err != nil {
		return nil, err
	}
	if env.Type != protocol.MsgTestStart {
		return nil, fmt.Errorf("expected test_start, got %s", env.Type)
	}

	duration := time.Duration(p.Duration) * time.Second
	deadline := time.Now().Add(duration)
	var totalBytes atomic.Int64
	var wg sync.WaitGroup

	if p.Reverse {
		// Server sends data to client.
		for i := 0; i < p.Streams; i++ {
			dataConn, err := ln.Accept()
			if err != nil {
				break
			}
			platform.TuneConn(dataConn, 0)
			wg.Add(1)
			go func(dc net.Conn) {
				defer wg.Done()
				defer dc.Close()
				buf := make([]byte, 128*1024)
				for time.Now().Before(deadline) {
					dc.SetWriteDeadline(deadline)
					n, err := dc.Write(buf)
					if err != nil {
						break
					}
					totalBytes.Add(int64(n))
				}
			}(dataConn)
		}
	} else {
		// Server receives data from client.
		for i := 0; i < p.Streams; i++ {
			dataConn, err := ln.Accept()
			if err != nil {
				break
			}
			platform.TuneConn(dataConn, 0)
			wg.Add(1)
			go func(dc net.Conn) {
				defer wg.Done()
				defer dc.Close()
				buf := make([]byte, 128*1024)
				for {
					dc.SetReadDeadline(deadline.Add(2 * time.Second))
					n, err := dc.Read(buf)
					if n > 0 {
						totalBytes.Add(int64(n))
					}
					if err != nil {
						break
					}
				}
			}(dataConn)
		}
	}

	wg.Wait()

	elapsed := duration.Seconds()
	total := totalBytes.Load()
	return TCPMetrics{
		Direction:  dirString(p.Reverse),
		Duration:   elapsed,
		BytesTotal: total,
		BitsPerSec: float64(total) * 8 / elapsed,
		Streams:    p.Streams,
	}, nil
}

// RunTCPClient runs the TCP throughput test from the client side.
func RunTCPClient(ctx context.Context, conn net.Conn, serverAddr string, streams, duration int, reverse bool, progress func(float64)) (*TCPMetrics, error) {
	if streams <= 0 {
		streams = 4
	}
	if duration <= 0 {
		duration = 10
	}

	params, _ := json.Marshal(protocol.TCPParams{
		Streams:  streams,
		Duration: duration,
		Reverse:  reverse,
	})
	if err := protocol.WriteMsg(conn, protocol.MsgTestReq, protocol.TestRequest{
		Test:   protocol.TestTCP,
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
	dataAddr := net.JoinHostPort(host, fmt.Sprintf("%d", ready.DataPort))

	// Signal start.
	if err := protocol.WriteMsg(conn, protocol.MsgTestStart, protocol.TestStart{}); err != nil {
		return nil, err
	}

	dur := time.Duration(duration) * time.Second
	deadline := time.Now().Add(dur)
	start := time.Now()
	var totalBytes atomic.Int64
	var wg sync.WaitGroup
	var intervals []IntervalStats

	if reverse {
		// Client receives from server.
		for i := 0; i < streams; i++ {
			dc, err := net.Dial("tcp", dataAddr)
			if err != nil {
				return nil, fmt.Errorf("data stream %d: %w", i, err)
			}
			platform.TuneConn(dc, 0)
			wg.Add(1)
			go func(c net.Conn) {
				defer wg.Done()
				defer c.Close()
				buf := make([]byte, 128*1024)
				for {
					c.SetReadDeadline(deadline.Add(2 * time.Second))
					n, err := c.Read(buf)
					if n > 0 {
						totalBytes.Add(int64(n))
					}
					if err != nil {
						break
					}
				}
			}(dc)
		}
	} else {
		// Client sends to server.
		for i := 0; i < streams; i++ {
			dc, err := net.Dial("tcp", dataAddr)
			if err != nil {
				return nil, fmt.Errorf("data stream %d: %w", i, err)
			}
			platform.TuneConn(dc, 0)
			wg.Add(1)
			go func(c net.Conn) {
				defer wg.Done()
				defer c.Close()
				buf := make([]byte, 128*1024)
				for time.Now().Before(deadline) {
					c.SetWriteDeadline(deadline)
					n, err := c.Write(buf)
					if err != nil {
						break
					}
					totalBytes.Add(int64(n))
				}
			}(dc)
		}
	}

	// Progress reporting in 1-second intervals.
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		lastBytes := int64(0)
		intervalStart := 0.0
		for {
			select {
			case <-ticker.C:
				cur := totalBytes.Load()
				elapsed := time.Since(start).Seconds()
				delta := cur - lastBytes
				bps := float64(delta) * 8 / (elapsed - intervalStart)
				intervals = append(intervals, IntervalStats{
					Start:      intervalStart,
					End:        elapsed,
					Bytes:      delta,
					BitsPerSec: bps,
				})
				intervalStart = elapsed
				lastBytes = cur
				if progress != nil {
					progress(bps)
				}
			case <-ctx.Done():
				return
			}
			if time.Now().After(deadline) {
				return
			}
		}
	}()

	wg.Wait()
	elapsed := time.Since(start).Seconds()
	total := totalBytes.Load()

	// Read server result.
	protocol.ReadMsg(conn)

	return &TCPMetrics{
		Direction:  dirString(reverse),
		Duration:   elapsed,
		BytesTotal: total,
		BitsPerSec: float64(total) * 8 / elapsed,
		Streams:    streams,
		Intervals:  intervals,
	}, nil
}

// RunTCPReceiveOnly is used by the server-side to handle reverse mode.
func RunTCPReceiveOnly(conn net.Conn, duration time.Duration) (int64, error) {
	deadline := time.Now().Add(duration)
	var total int64
	buf := make([]byte, 128*1024)
	for {
		conn.SetReadDeadline(deadline.Add(2 * time.Second))
		n, err := conn.Read(buf)
		if n > 0 {
			total += int64(n)
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			break
		}
	}
	return total, nil
}

func dirString(reverse bool) string {
	if reverse {
		return "download"
	}
	return "upload"
}

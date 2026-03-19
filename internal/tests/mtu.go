package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"time"

	"github.com/goozt/gospeed/internal/platform"
	"github.com/goozt/gospeed/internal/protocol"
	"github.com/goozt/gospeed/internal/server"
)

func init() {
	server.RegisterHandler(protocol.TestMTU, handleMTUServer)
}

// MTU server: receive UDP packets of varying sizes and report what arrived.
func handleMTUServer(ctx context.Context, conn net.Conn, params json.RawMessage) (any, error) {
	var p protocol.MTUParams
	if params != nil {
		json.Unmarshal(params, &p)
	}
	if p.MinSize <= 0 {
		p.MinSize = 576
	}
	if p.MaxSize <= 0 {
		p.MaxSize = 9000
	}

	// Open a UDP listener on an ephemeral port.
	udpAddr, err := net.ResolveUDPAddr("udp", ":0")
	if err != nil {
		return nil, err
	}
	udpConn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return nil, err
	}
	defer udpConn.Close()

	port := udpConn.LocalAddr().(*net.UDPAddr).Port

	// Tell client which port to send to.
	if err := protocol.WriteMsg(conn, protocol.MsgTestReady, protocol.TestReady{DataPort: port}); err != nil {
		return nil, err
	}

	// Receive loop: report the size of each received packet.
	buf := make([]byte, 10000)
	maxReceived := 0
	for {
		select {
		case <-ctx.Done():
			return MTUMetrics{MTU: maxReceived}, nil
		default:
		}

		udpConn.SetReadDeadline(time.Now().Add(5 * time.Second))
		n, _, err := udpConn.ReadFromUDP(buf)
		if err != nil {
			break
		}
		if n == 0 {
			break
		}
		// Packet with only header byte 0xFF means "end of test".
		if n == 1 && buf[0] == 0xFF {
			break
		}
		if n > maxReceived {
			maxReceived = n
		}
		// Echo back the received size.
		if err := protocol.WriteMsg(conn, protocol.MsgTestStart, struct {
			Size int `json:"size"`
		}{Size: n}); err != nil {
			break
		}
	}

	return MTUMetrics{MTU: maxReceived + 28}, nil // +28 for IP(20)+UDP(8) headers
}

// RunMTUClient runs the path MTU discovery test from the client side.
func RunMTUClient(ctx context.Context, conn net.Conn, serverAddr string) (*MTUMetrics, error) {
	params, _ := json.Marshal(protocol.MTUParams{MinSize: 576, MaxSize: 9000})
	if err := protocol.WriteMsg(conn, protocol.MsgTestReq, protocol.TestRequest{
		Test:   protocol.TestMTU,
		Params: params,
	}); err != nil {
		return nil, err
	}

	// Wait for ready with data port.
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

	// Resolve server host for UDP.
	host, _, err := net.SplitHostPort(serverAddr)
	if err != nil {
		host = serverAddr
	}
	udpAddr := fmt.Sprintf("%s:%d", host, ready.DataPort)
	raddr, err := net.ResolveUDPAddr("udp", udpAddr)
	if err != nil {
		return nil, err
	}
	udpConn, err := net.DialUDP("udp", nil, raddr)
	if err != nil {
		return nil, err
	}
	defer udpConn.Close()

	// Set DF bit.
	platform.SetDontFragment(udpConn)

	// Binary search for largest packet that gets through.
	low, high := 548, 8972 // payload sizes (MTU - 28 for IP+UDP headers)
	lastGood := low

	for low <= high {
		select {
		case <-ctx.Done():
			return &MTUMetrics{MTU: lastGood + 28}, nil
		default:
		}

		mid := (low + high) / 2
		payload := make([]byte, mid)
		// Send the probe.
		_, err := udpConn.Write(payload)
		if err != nil {
			// Likely "message too long" — packet too big.
			high = mid - 1
			continue
		}

		// Wait for server confirmation with a short deadline so we don't
		// hang forever when the probe packet is silently dropped.
		conn.SetDeadline(time.Now().Add(3 * time.Second))
		env, err := protocol.ReadMsg(conn)
		conn.SetDeadline(time.Time{}) // clear deadline
		if err != nil {
			high = mid - 1
			continue
		}
		if env.Type == protocol.MsgTestStart {
			var resp struct {
				Size int `json:"size"`
			}
			protocol.DecodeBody(env, &resp)
			if resp.Size >= mid {
				lastGood = mid
				low = mid + 1
			} else {
				high = mid - 1
			}
		} else {
			high = mid - 1
		}
	}

	// Signal end of test.
	udpConn.Write([]byte{0xFF})

	// Read server's test result.
	protocol.ReadMsg(conn)

	return &MTUMetrics{MTU: lastGood + 28}, nil
}

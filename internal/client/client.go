package client

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/goozt/gospeed/internal/protocol"
	"github.com/goozt/gospeed/internal/results"
	"github.com/goozt/gospeed/internal/tests"
)

// Config holds client configuration.
type Config struct {
	Server   string
	Tests    []protocol.TestType
	JSON     bool
	CSV      bool
	History  bool
	Ping     bool
	TLS      bool
	TLSSkip  bool
	Streams  int
	Duration int
}

// Client manages a connection to a gospeed server.
type Client struct {
	cfg         Config
	conn        net.Conn
	sessID      string
	serverTests []protocol.TestType
	progress    *Progress
}

// New creates a new client with the given configuration.
func New(cfg Config) *Client {
	return &Client{
		cfg:      cfg,
		progress: NewProgress(cfg.JSON),
	}
}

// Run connects to the server, runs tests, and outputs results.
func (c *Client) Run(ctx context.Context) error {
	if c.cfg.History {
		return results.PrintHistory(os.Stdout, 20)
	}

	go func() {
		<-ctx.Done()
		if c.conn != nil {
			c.conn.SetDeadline(time.Now())
		}
	}()

	if c.cfg.Ping {
		return c.runPing(ctx)
	}

	if err := c.connect(ctx); err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer c.close()

	testList := c.filterTests()
	report := c.executeTests(ctx, testList)

	if len(report.Results) > 0 {
		report.OverallGrade = results.ComputeOverallGrade(report.Results)
		c.outputReport(report)
	}

	return nil
}

func (c *Client) runPing(ctx context.Context) error {
	const maxRetries = 5
	for attempt := range maxRetries {
		err := c.connect(ctx)
		if err == nil {
			fmt.Printf("OK — server %s is reachable (session %s)\n", c.cfg.Server, c.sessID)
			c.close()
			return nil
		}
		c.close()
		if attempt < maxRetries-1 {
			fmt.Fprintf(os.Stderr, "ping attempt %d/%d failed: %v\n", attempt+1, maxRetries, err)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(2 * time.Second):
			}
		} else {
			return fmt.Errorf("ping failed after %d attempts: %w", maxRetries, err)
		}
	}
	return nil
}

func (c *Client) filterTests() []protocol.TestType {
	testList := c.cfg.Tests
	if len(testList) == 0 {
		testList = protocol.DefaultTests
	}

	if len(c.serverTests) == 0 {
		return testList
	}

	supported := make(map[protocol.TestType]bool, len(c.serverTests))
	for _, t := range c.serverTests {
		supported[t] = true
	}
	filtered := testList[:0:0]
	for _, t := range testList {
		if supported[t] {
			filtered = append(filtered, t)
		} else {
			fmt.Fprintf(os.Stderr, "  skipping %s: not supported by server\n", t)
		}
	}
	return filtered
}

func (c *Client) executeTests(ctx context.Context, testList []protocol.TestType) *results.Report {
	report := &results.Report{
		Timestamp: time.Now(),
		Server:    c.cfg.Server,
	}

	for _, t := range testList {
		if ctx.Err() != nil {
			fmt.Fprintf(os.Stderr, "\naborted.\n")
			break
		}
		result, err := c.runTest(ctx, t)
		if err != nil {
			if ctx.Err() != nil {
				fmt.Fprintf(os.Stderr, "\naborted.\n")
				break
			}
			fmt.Fprintf(os.Stderr, "  %s test failed: %v\n", t, err)
			continue
		}
		if result != nil {
			report.Results = append(report.Results, *result)
		}
	}
	return report
}

func (c *Client) outputReport(report *results.Report) {
	switch {
	case c.cfg.JSON:
		results.FormatJSON(os.Stdout, report)
	case c.cfg.CSV:
		results.FormatCSV(os.Stdout, report)
	default:
		c.progress.Clear()
		fmt.Println()
		results.FormatTable(os.Stdout, report)
	}
	if err := results.SaveHistory(report); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to save history: %v\n", err)
	}
}

func (c *Client) connect(ctx context.Context) error {
	// Append default port if missing.
	addr := c.cfg.Server
	if _, _, err := net.SplitHostPort(addr); err != nil {
		addr = net.JoinHostPort(addr, "9000")
	}

	dialer := &net.Dialer{Timeout: 10 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return err
	}
	c.conn = conn

	// Generate client ID.
	var idBytes [8]byte
	rand.Read(idBytes[:])
	clientID := hex.EncodeToString(idBytes[:])

	// Handshake.
	if err := protocol.WriteMsg(conn, protocol.MsgHello, protocol.Hello{
		Version:  protocol.ProtocolVersion,
		ClientID: clientID,
	}); err != nil {
		return fmt.Errorf("send hello: %w", err)
	}

	env, err := protocol.ReadMsg(conn)
	if err != nil {
		return fmt.Errorf("read hello_ack: %w", err)
	}
	if env.Type == protocol.MsgError {
		var e protocol.ErrorMsg
		protocol.DecodeBody(env, &e)
		return fmt.Errorf("server error: %s", e.Message)
	}
	if env.Type != protocol.MsgHelloAck {
		return fmt.Errorf("expected hello_ack, got %s", env.Type)
	}

	var ack protocol.HelloAck
	protocol.DecodeBody(env, &ack)
	c.sessID = ack.SessionID
	c.serverTests = ack.Tests

	return nil
}

func (c *Client) close() {
	if c.conn != nil {
		protocol.WriteMsg(c.conn, protocol.MsgGoodbye, protocol.Goodbye{})
		c.conn.Close()
	}
}

func (c *Client) runTest(ctx context.Context, t protocol.TestType) (result *results.TestResult, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
			result = nil
		}
	}()

	timeout := max(time.Duration(c.cfg.Duration*3)*time.Second, 30*time.Second)
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Enforce the timeout at the connection level so blocking reads/writes
	// are interrupted when the per-test context expires (not on normal cancel).
	deadline := time.Now().Add(timeout)
	c.conn.SetDeadline(deadline)
	defer c.conn.SetDeadline(time.Time{}) // clear deadline for next test

	c.progress.TestStart(string(t))

	metrics, grade, err := c.dispatchTest(ctx, t)
	if err != nil {
		return nil, err
	}

	c.progress.Clear()
	return &results.TestResult{
		Test:    string(t),
		Metrics: metrics,
		Grade:   grade,
	}, nil
}

func (c *Client) dispatchTest(ctx context.Context, t protocol.TestType) (any, results.Grade, error) {
	switch t {
	case protocol.TestLatency:
		m, e := tests.RunLatencyClient(ctx, c.conn, 20)
		if e != nil {
			return nil, "", e
		}
		return m, results.GradeLatency(m.Avg), nil
	case protocol.TestMTU:
		m, e := tests.RunMTUClient(ctx, c.conn, c.cfg.Server)
		return m, "", e
	case protocol.TestTCP:
		return c.runTCPTest(ctx)
	case protocol.TestUDP:
		return c.runUDPTest(ctx)
	case protocol.TestJitter:
		m, e := tests.RunJitterClient(ctx, c.conn, c.cfg.Server, 20, 200)
		if e != nil {
			return nil, "", e
		}
		return m, results.GradeJitter(m.AvgJitter), nil
	case protocol.TestBufferbloat:
		return c.runBufferbloatTest(ctx)
	case protocol.TestDNS:
		host, _, _ := net.SplitHostPort(c.cfg.Server)
		m, e := tests.RunDNSClient(ctx, host, 10)
		return m, "", e
	case protocol.TestConnect:
		m, e := tests.RunConnectClient(ctx, c.cfg.Server, 10)
		return m, "", e
	case protocol.TestBidir:
		m, e := tests.RunBidirClient(ctx, c.conn, c.cfg.Server, c.cfg.Streams, c.cfg.Duration, func(dir string, bps float64) {
			c.progress.Status("Bidir %s: %s", dir, FormatBPS(bps))
		})
		return m, "", e
	default:
		return nil, "", fmt.Errorf("unknown test: %s", t)
	}
}

func (c *Client) runTCPTest(ctx context.Context) (any, results.Grade, error) {
	m, e := tests.RunTCPClient(ctx, c.conn, c.cfg.Server, c.cfg.Streams, c.cfg.Duration, false, func(bps float64) {
		c.progress.Status("TCP upload: %s", FormatBPS(bps))
	})
	if e != nil {
		return nil, "", e
	}
	return m, results.GradeThroughput(m.BitsPerSec), nil
}

func (c *Client) runUDPTest(ctx context.Context) (any, results.Grade, error) {
	m, e := tests.RunUDPClient(ctx, c.conn, c.cfg.Server, c.cfg.Duration, 1400, 0, func(bps float64) {
		c.progress.Status("UDP: %s", FormatBPS(bps))
	})
	if e != nil {
		return nil, "", e
	}
	return m, results.GradeLoss(m.LossPercent), nil
}

func (c *Client) runBufferbloatTest(ctx context.Context) (any, results.Grade, error) {
	m, e := tests.RunBufferbloatClient(ctx, c.conn, c.cfg.Server, c.cfg.Duration, c.cfg.Streams, func(status string) {
		c.progress.Status("Bufferbloat: %s", status)
	})
	if e != nil {
		return nil, "", e
	}
	return m, results.GradeBufferbloat(m.RPM), nil
}

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
	completed   []results.TestResult // results from completed tests
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

	// Print header immediately for interactive (table) output.
	if !c.cfg.JSON && !c.cfg.CSV {
		c.progress.Clear()
		fmt.Println()
		fmt.Fprintln(os.Stdout, results.Header("gospeed results"))
		fmt.Fprintf(os.Stdout, "  Server: %s\n", results.Bold(c.cfg.Server))
		fmt.Fprintf(os.Stdout, "  Time:   %s\n\n", results.Dim(time.Now().Format(time.RFC3339)))
	}

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
		if supported[t] || isClientOnly(t) {
			filtered = append(filtered, t)
		} else {
			fmt.Fprintf(os.Stderr, "  skipping %s: not supported by server\n", t)
		}
	}
	return filtered
}

func isClientOnly(t protocol.TestType) bool {
	return t == protocol.TestDNS || t == protocol.TestConnect || t == protocol.TestBidir
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
			c.completed = append(c.completed, *result)
			// Print result immediately for interactive output.
			if !c.cfg.JSON && !c.cfg.CSV {
				results.FormatTestResult(os.Stdout, *result)
				fmt.Println()
			}
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
		// Results already printed incrementally; just print the overall grade.
		fmt.Fprintf(os.Stdout, "%s\n", results.Header(fmt.Sprintf("Overall: %s", results.ColorGrade(report.OverallGrade))))
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
		c.cfg.Server = addr // store normalized address so tests can parse host:port
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

	mult := 3
	if t == protocol.TestBufferbloat {
		mult = 5 // extra time for baseline latency phase
	}
	timeout := max(time.Duration(c.cfg.Duration*mult)*time.Second, 30*time.Second)
	testCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	c.progress.TestStart(string(t))

	// Run the test in a goroutine so we can enforce the timeout even when
	// a blocking read/write on the TCP connection ignores the context.
	type testResult struct {
		metrics any
		grade   results.Grade
		err     error
	}
	ch := make(chan testResult, 1)
	go func() {
		m, g, e := c.dispatchTest(testCtx, t)
		ch <- testResult{m, g, e}
	}()

	select {
	case res := <-ch:
		if res.err != nil {
			return nil, res.err
		}
		c.progress.Clear()
		return &results.TestResult{
			Test:    string(t),
			Metrics: res.metrics,
			Grade:   res.grade,
		}, nil

	case <-testCtx.Done():
		// Test timed out — a goroutine is likely stuck on a blocking TCP read.
		// Force-close the connection to unblock it, then reconnect.
		c.conn.Close()
		<-ch // wait for goroutine to exit
		if ctx.Err() == nil {
			// Parent context is still valid; reconnect for subsequent tests.
			if reconnErr := c.connect(ctx); reconnErr != nil {
				return nil, fmt.Errorf("%s timed out, reconnect failed: %w", t, reconnErr)
			}
		}
		return nil, fmt.Errorf("%s timed out", t)
	}
}

func (c *Client) dispatchTest(ctx context.Context, t protocol.TestType) (any, results.Grade, error) {
	switch t {
	case protocol.TestLatency:
		return c.runLatencyTest(ctx)
	case protocol.TestMTU:
		return c.runMTUTest(ctx)
	case protocol.TestTCP:
		return c.runTCPTest(ctx)
	case protocol.TestUDP:
		return c.runUDPTest(ctx)
	case protocol.TestJitter:
		return c.runJitterTest(ctx)
	case protocol.TestBufferbloat:
		return c.runBufferbloatTest(ctx)
	case protocol.TestDNS:
		return c.runDNSTest(ctx)
	case protocol.TestConnect:
		return c.runConnectTest(ctx)
	case protocol.TestBidir:
		return c.runBidirTest(ctx)
	default:
		return nil, "", fmt.Errorf("unknown test: %s", t)
	}
}

func (c *Client) runLatencyTest(ctx context.Context) (any, results.Grade, error) {
	m, e := tests.RunLatencyClient(ctx, c.conn, 20)
	if e != nil {
		return nil, "", e
	}
	return m, results.GradeLatency(m.Avg), nil
}

func (c *Client) runMTUTest(ctx context.Context) (any, results.Grade, error) {
	m, e := tests.RunMTUClient(ctx, c.conn, c.cfg.Server)
	if e != nil {
		return nil, "", e
	}
	return m, results.GradeMTU(m.MTU), nil
}

func (c *Client) runJitterTest(ctx context.Context) (any, results.Grade, error) {
	m, e := tests.RunJitterClient(ctx, c.conn, c.cfg.Server, 20, 200)
	if e != nil {
		return nil, "", e
	}
	return m, results.GradeJitter(m.AvgJitter), nil
}

func (c *Client) runDNSTest(ctx context.Context) (any, results.Grade, error) {
	host, _, _ := net.SplitHostPort(c.cfg.Server)
	m, e := tests.RunDNSClient(ctx, host, 10)
	if e != nil {
		return nil, "", e
	}
	return m, results.GradeDNS(m.Avg), nil
}

func (c *Client) runConnectTest(ctx context.Context) (any, results.Grade, error) {
	m, e := tests.RunConnectClient(ctx, c.cfg.Server, 10)
	if e != nil {
		return nil, "", e
	}
	return m, results.GradeConnect(m.Avg), nil
}

func (c *Client) runBidirTest(ctx context.Context) (any, results.Grade, error) {
	m, e := tests.RunBidirClient(ctx, c.conn, c.cfg.Server, c.cfg.Streams, c.cfg.Duration, func(dir string, bps float64) {
		c.progress.Status("Bidir %s: %s", dir, FormatBPS(bps))
	})
	if e != nil {
		return nil, "", e
	}
	worse := min(m.Upload.BitsPerSec, m.Download.BitsPerSec)
	return m, results.GradeThroughput(worse), nil
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

// maxThroughputBps extracts the highest bits-per-second value from a test metric.
func maxThroughputBps(metrics any) float64 {
	switch m := metrics.(type) {
	case *tests.TCPMetrics:
		return m.BitsPerSec
	case tests.TCPMetrics:
		return m.BitsPerSec
	case *tests.BidirMetrics:
		return max(m.Upload.BitsPerSec, m.Download.BitsPerSec)
	case tests.BidirMetrics:
		return max(m.Upload.BitsPerSec, m.Download.BitsPerSec)
	case *tests.BufferbloatMetrics:
		return m.Throughput.BitsPerSec
	case tests.BufferbloatMetrics:
		return m.Throughput.BitsPerSec
	default:
		return 0
	}
}

// estimateUDPBandwidth derives a UDP send rate from prior test results.
// Uses TCP throughput if available, otherwise falls back to 100 Mbps.
func (c *Client) estimateUDPBandwidth() int64 {
	const fallback = 100_000_000 // 100 Mbps

	var maxBps float64
	for _, r := range c.completed {
		if bps := maxThroughputBps(r.Metrics); bps > maxBps {
			maxBps = bps
		}
	}

	if maxBps > 0 {
		// Ceil to nearest 100 Mbps (e.g., 189 Mbps → 200 Mbps).
		mbps := int64(maxBps / 1_000_000)
		rounded := ((mbps + 99) / 100) * 100
		return rounded * 1_000_000
	}
	return fallback
}

func (c *Client) runUDPTest(ctx context.Context) (any, results.Grade, error) {
	bandwidth := c.estimateUDPBandwidth()
	m, e := tests.RunUDPClient(ctx, c.conn, c.cfg.Server, c.cfg.Duration, 1400, bandwidth, func(bps float64) {
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

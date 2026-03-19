package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/goozt/gospeed/internal/client"
	"github.com/goozt/gospeed/internal/protocol"
	"github.com/goozt/gospeed/internal/version"
)

func main() {
	server := flag.String("server", "", "server address (host:port)")
	jsonOut := flag.Bool("json", false, "output results as JSON")
	csvOut := flag.Bool("csv", false, "output results as CSV")
	history := flag.Bool("history", false, "show test history")
	streams := flag.Int("streams", 4, "number of parallel streams for throughput tests")
	duration := flag.Int("duration", 10, "test duration in seconds")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.StringVar(server, "s", "", "server address (host:port)")
	flag.BoolVar(history, "h", false, "show test history")
	flag.IntVar(streams, "t", 4, "number of parallel streams for throughput tests")
	flag.IntVar(duration, "d", 10, "test duration in seconds")
	flag.BoolVar(showVersion, "v", false, "print version and exit")

	// Individual test flags.
	doLatency := flag.Bool("latency", false, "run latency test")
	doMTU := flag.Bool("mtu", false, "run path MTU discovery")
	doTCP := flag.Bool("tcp", false, "run TCP throughput test")
	doUDP := flag.Bool("udp", false, "run UDP throughput test")
	doJitter := flag.Bool("jitter", false, "run jitter test")
	doBloat := flag.Bool("bufferbloat", false, "run bufferbloat detection")
	doDNS := flag.Bool("dns", false, "run DNS resolution test")
	doConnect := flag.Bool("connect", false, "run TCP connect time test")
	doBidir := flag.Bool("bidir", false, "run bidirectional throughput test")
	doAll := flag.Bool("all", false, "run all tests")
	flag.BoolVar(doAll, "a", false, "run all tests")

	// TLS flags.
	useTLS := flag.Bool("tls", false, "use TLS connection")
	tlsSkip := flag.Bool("tls-skip-verify", false, "skip TLS certificate verification")

	flag.Parse()

	if *showVersion {
		fmt.Printf("gospeed %s (%s) built %s\n", version.Version, version.Commit, version.Date)
		fmt.Printf("  go: %s, os/arch: %s/%s\n", runtime.Version(), runtime.GOOS, runtime.GOARCH)
		os.Exit(0)
	}

	// Resolve server address.
	addr := *server
	if addr == "" {
		addr = os.Getenv("GOSPEED_SERVER_ADDR")
	}
	if addr == "" {
		addr = protocol.DefaultAddr
	}

	// Build test list.
	var testList []protocol.TestType
	if *doAll {
		testList = protocol.AllTests
	} else {
		if *doLatency {
			testList = append(testList, protocol.TestLatency)
		}
		if *doMTU {
			testList = append(testList, protocol.TestMTU)
		}
		if *doTCP {
			testList = append(testList, protocol.TestTCP)
		}
		if *doUDP {
			testList = append(testList, protocol.TestUDP)
		}
		if *doJitter {
			testList = append(testList, protocol.TestJitter)
		}
		if *doBloat {
			testList = append(testList, protocol.TestBufferbloat)
		}
		if *doDNS {
			testList = append(testList, protocol.TestDNS)
		}
		if *doConnect {
			testList = append(testList, protocol.TestConnect)
		}
		if *doBidir {
			testList = append(testList, protocol.TestBidir)
		}
	}

	_ = useTLS
	_ = tlsSkip

	cfg := client.Config{
		Server:   addr,
		Tests:    testList,
		JSON:     *jsonOut,
		CSV:      *csvOut,
		History:  *history,
		TLS:      *useTLS,
		TLSSkip:  *tlsSkip,
		Streams:  *streams,
		Duration: *duration,
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	c := client.New(cfg)
	if err := c.Run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

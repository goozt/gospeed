package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/goozt/gospeed/internal/server"
	"github.com/goozt/gospeed/internal/version"

	// Register all test handlers.
	_ "github.com/goozt/gospeed/internal/tests"
)

func main() {
	addr := flag.String("addr", "", "listen address")
	host := flag.String("host", "", "specific host address")
	port := flag.Int("port", 9000, "listening port")
	tlsCert := flag.String("tls-cert", "", "TLS certificate file")
	tlsKey := flag.String("tls-key", "", "TLS key file")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.StringVar(addr, "a", "", "print version and exit (shorthand)")
	flag.StringVar(host, "h", "", "print version and exit (shorthand)")
	flag.IntVar(port, "p", 9000, "print version and exit (shorthand)")
	flag.BoolVar(showVersion, "v", false, "print version and exit (shorthand)")
	flag.Parse()

	if *showVersion {
		fmt.Printf("gospeed-server %s (%s) built %s\n", version.Version, version.Commit, version.Date)
		fmt.Printf("  go: %s, os/arch: %s/%s\n", runtime.Version(), runtime.GOOS, runtime.GOARCH)
		os.Exit(0)
	}

	_ = tlsCert
	_ = tlsKey

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if *addr == "" {
		*addr = fmt.Sprintf("%s:%d", *host, *port)
	}

	srv := server.New(*addr)
	if err := srv.ListenAndServe(ctx); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

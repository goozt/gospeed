package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/goozt/gospeed/internal/server"
	"github.com/goozt/gospeed/internal/tlsutil"
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
	tlsSelfSigned := flag.Bool("tls-self-signed", false, "use auto-generated self-signed certificate")
	tlsACME := flag.Bool("tls-acme", false, "use Let's Encrypt ACME for certificate provisioning")
	domain := flag.String("domain", "", "domain name for ACME certificate")
	email := flag.String("email", "", "email address for ACME account")
	certDir := flag.String("cert-dir", "", "directory to cache ACME certificates")
	healthPort := flag.Int("health", 0, "start HTTP health check server on given port")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.StringVar(addr, "a", "", "listen address (shorthand)")
	flag.StringVar(host, "h", "", "specific host address (shorthand)")
	flag.IntVar(port, "p", 9000, "listening port (shorthand)")
	flag.BoolVar(showVersion, "v", false, "print version and exit (shorthand)")
	flag.Parse()

	if *showVersion {
		fmt.Printf("gospeed-server %s (%s) built %s\n", version.Version, version.Commit, version.Date)
		fmt.Printf("  go: %s, os/arch: %s/%s\n", runtime.Version(), runtime.GOOS, runtime.GOARCH)
		os.Exit(0)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if *addr == "" {
		*addr = fmt.Sprintf("%s:%d", *host, *port)
	}

	srv := server.New(*addr)

	if *healthPort > 0 {
		go func() {
			mux := http.NewServeMux()
			mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{"status":"ok"}`))
			})
			hAddr := fmt.Sprintf(":%d", *healthPort)
			log.Printf("health check listening on %s", hAddr)
			if err := http.ListenAndServe(hAddr, mux); err != nil {
				log.Printf("health server error: %v", err)
			}
		}()
	}

	switch {
	case *tlsACME:
		tlsCfg, err := tlsutil.ACMEConfig(*domain, *email, *certDir)
		if err != nil {
			log.Fatalf("acme: %v", err)
		}
		if err := srv.ListenAndServeTLS(ctx, tlsCfg); err != nil {
			log.Fatalf("server error: %v", err)
		}
	case *tlsSelfSigned:
		cert, err := tlsutil.SelfSignedCert()
		if err != nil {
			log.Fatalf("self-signed cert: %v", err)
		}
		tlsCfg := &tls.Config{Certificates: []tls.Certificate{cert}}
		if err := srv.ListenAndServeTLS(ctx, tlsCfg); err != nil {
			log.Fatalf("server error: %v", err)
		}
	case *tlsCert != "" && *tlsKey != "":
		cert, err := tls.LoadX509KeyPair(*tlsCert, *tlsKey)
		if err != nil {
			log.Fatalf("load tls cert: %v", err)
		}
		tlsCfg := &tls.Config{Certificates: []tls.Certificate{cert}}
		if err := srv.ListenAndServeTLS(ctx, tlsCfg); err != nil {
			log.Fatalf("server error: %v", err)
		}
	default:
		if err := srv.ListenAndServe(ctx); err != nil {
			log.Fatalf("server error: %v", err)
		}
	}
}

package protocol

import (
	"crypto/tls"
	"fmt"
	"net"
)

// TLSListener wraps a TCP listener with TLS.
func TLSListener(ln net.Listener, certFile, keyFile string) (net.Listener, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("load TLS cert: %w", err)
	}
	cfg := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}
	return tls.NewListener(ln, cfg), nil
}

// TLSDial connects to addr with TLS.
func TLSDial(addr string, skipVerify bool) (net.Conn, error) {
	cfg := &tls.Config{
		InsecureSkipVerify: skipVerify,
		MinVersion:         tls.VersionTLS12,
	}
	return tls.Dial("tcp", addr, cfg)
}

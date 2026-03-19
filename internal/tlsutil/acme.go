package tlsutil

import (
	"crypto/tls"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/crypto/acme/autocert"
)

// ACMEConfig returns a *tls.Config that provisions certificates via Let's Encrypt.
func ACMEConfig(domain, email, cacheDir string) (*tls.Config, error) {
	if domain == "" {
		return nil, fmt.Errorf("--domain is required when using --tls-acme")
	}
	if email == "" {
		return nil, fmt.Errorf("--email is required when using --tls-acme")
	}

	if cacheDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("determine home directory: %w", err)
		}
		cacheDir = filepath.Join(home, ".gospeed", "certs")
	}

	m := &autocert.Manager{
		Cache:      autocert.DirCache(cacheDir),
		Prompt:     autocert.AcceptTOS,
		Email:      email,
		HostPolicy: autocert.HostWhitelist(domain),
	}

	return m.TLSConfig(), nil
}

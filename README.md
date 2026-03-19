<p align="center">
  <img src="docs/assets/gospeed_front_banner.png" alt="gospeed" width="100%">
</p>

<p align="center">
  <a href="https://github.com/goozt/gospeed/actions/workflows/ci.yml"><img src="https://github.com/goozt/gospeed/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
  <a href="https://codecov.io/gh/goozt/gospeed"><img src="https://codecov.io/gh/goozt/gospeed/graph/badge.svg?token=1R5W9MDU6Y" alt="codecov"></a>
  <a href="https://goreportcard.com/report/github.com/goozt/gospeed"><img src="https://goreportcard.com/badge/github.com/goozt/gospeed" alt="Go Report Card"></a>
  <a href="https://pkg.go.dev/github.com/goozt/gospeed"><img src="https://pkg.go.dev/badge/github.com/goozt/gospeed.svg" alt="Go Reference"></a>
  <a href="https://opensource.org/licenses/MIT"><img src="https://img.shields.io/badge/License-MIT-blue.svg" alt="License: MIT"></a>
  <a href="https://github.com/goozt/gospeed/releases/latest"><img src="https://img.shields.io/github/v/release/goozt/gospeed" alt="Release"></a>
</p>

A fast, zero-dependency network speed testing tool written in Go. Client-server architecture for accurate network performance measurement.

## Features

- **9 test types**: TCP/UDP throughput, latency, jitter, bufferbloat (RPM), path MTU discovery, DNS resolution, TCP connect time, bidirectional throughput
- **Scientifically grounded**: Based on RFC 6349, RFC 2544, ITU-T Y.1564
- **Multi-client server**: Handles concurrent clients
- **Quality grades**: A-F ratings for each metric with color-coded terminal output
- **Multiple output formats**: Table (default), JSON (`--json`), CSV (`--csv`)
- **Test history**: Local JSONL storage with trend tracking (`--history`)
- **Cross-platform**: Native Windows, macOS, Linux, FreeBSD support
- **Zero dependencies**: Pure Go standard library — no external packages
- **TLS support**: Optional encrypted connections
- **Docker ready**: Multi-stage Dockerfile and docker-compose included

## Installation

### Quick install

```sh
# Linux / macOS / FreeBSD
curl -fsSL https://gospeed.goozt.org/install.sh | bash

# Include server binary
curl -fsSL https://gospeed.goozt.org/install.sh | bash -s -- --server
```

```powershell
# Windows (PowerShell)
irm https://gospeed.goozt.org/install.ps1 | iex
```

### Binary download

Download pre-built binaries from the [releases page](https://github.com/goozt/gospeed/releases).

### Go install

```sh
go install github.com/goozt/gospeed/cmd/gospeed@latest
go install github.com/goozt/gospeed/cmd/gospeed-server@latest
```

### Docker (server only)

```sh
docker compose up -d
```

## Usage

### Server

```sh
# Start server on default port 9000
gospeed-server

# Custom address
gospeed-server --addr :8080

# With TLS (self-signed, auto-generated)
gospeed-server --tls-self-signed

# With TLS (your own certificate)
gospeed-server --tls-cert cert.pem --tls-key key.pem

# With TLS (Let's Encrypt ACME)
gospeed-server --tls-acme --domain speed.example.com --email admin@example.com
```

#### Server Options

```
-a, --addr string       Listen address (host:port)
-h, --host string       Specific host address
-p, --port int          Listening port (default 9000)
-v, --version           Print version and exit
    --tls-cert string   TLS certificate file
    --tls-key string    TLS key file
    --tls-self-signed   Use auto-generated self-signed certificate
    --tls-acme          Use Let's Encrypt ACME for certificate provisioning
    --domain string     Domain name for ACME certificate (required with --tls-acme)
    --email string      Email address for ACME account (required with --tls-acme)
    --cert-dir string   Directory to cache ACME certificates
    --health int        Start HTTP health check server on given port
```

### Client

```sh
# Run default test suite against a server
gospeed -s speed.example.com

# Use environment variable instead of -s flag
export GOSPEED_SERVER_ADDR=speed.example.com:9000
gospeed

# Default: localhost:9000
gospeed
```

### Test selection

```sh
# Run specific tests
gospeed -s host:9000 --latency
gospeed -s host:9000 --tcp --udp
gospeed -s host:9000 --bufferbloat

# Run all tests
gospeed -s host:9000 --all

# Default test suite (when no test flags given):
# MTU → Latency → TCP Throughput → Bufferbloat → UDP+Loss → Jitter
```

### Output formats

```sh
# JSON output (for scripting)
gospeed -s host:9000 --json

# CSV output
gospeed -s host:9000 --csv

# View test history with trends
gospeed --history
```

### Options

```
-s, --server string   Server address (host:port)
-t, --streams int     Number of parallel streams (default 4)
-d, --duration int    Test duration in seconds (default 10)
-h, --history         Show previous results with trends
-v, --version         Print version and exit
    --json            Output as JSON
    --csv             Output as CSV
    --tls             Use TLS connection
    --tls-skip-verify Skip TLS certificate verification
    --ping            Ping server before tests to check connectivity and exits

Test flags:
    --latency         Unloaded latency (RTT)
    --mtu             Path MTU discovery
    --tcp             TCP throughput
    --udp             UDP throughput + packet loss
    --jitter          Jitter measurement
    --bufferbloat     Bufferbloat detection (RPM)
    --dns             DNS resolution latency
    --connect         TCP connection setup time
    --bidir           Bidirectional throughput
-a, --all             Run all tests
```

## Test descriptions

| Test | Protocol | What it measures | Key metric |
|------|----------|------------------|------------|
| Latency | TCP | Round-trip time (20 samples) | min/avg/p95/max ms |
| MTU | UDP (DF bit) | Path Maximum Transmission Unit | bytes |
| TCP Throughput | TCP | Bandwidth with BDP-aware buffers | Mbps/Gbps |
| UDP Throughput | UDP | Bandwidth + packet loss + reordering | Mbps, loss % |
| Jitter | UDP | Inter-packet delay variation (RFC 3550) | ms |
| Bufferbloat | TCP+UDP | Latency under load → responsiveness | RPM |
| DNS | UDP/53 | DNS resolution latency | ms |
| Connect | TCP | SYN→established handshake time | ms |
| Bidirectional | TCP | Simultaneous upload + download | Mbps each |

## Grading

| Metric | A | B | C | D | F |
|--------|---|---|---|---|---|
| Latency | <20ms | <50ms | <100ms | <200ms | ≥200ms |
| Packet Loss | <0.1% | <0.5% | <1% | <2.5% | ≥2.5% |
| Jitter | <5ms | <10ms | <20ms | <50ms | ≥50ms |
| Bufferbloat (RPM) | ≥400 | ≥200 | ≥100 | ≥50 | <50 |
| Throughput | ≥100Mbps | ≥50Mbps | ≥25Mbps | ≥10Mbps | <10Mbps |

## Comparison with iperf3

| Feature | gospeed | iperf3 |
|---------|---------|--------|
| Concurrent clients | Yes | No (one at a time) |
| Windows support | Native | Cygwin-dependent |
| JSON output | Reliable | Broken with parallel/bidir |
| Bufferbloat detection | Built-in (RPM) | Not available |
| Unified metrics | Single run | Separate tools needed |
| Quality grading | A-F grades | Raw numbers only |
| Test history | Built-in trends | Not available |
| NAT-friendly | All client-initiated | Requires reverse mode |
| Dependencies | Zero | libssl, etc. |

## Architecture

```
┌─────────────┐     Control (TCP, JSON-framed)       ┌────────────────┐
│   gospeed   │◄────────────────────────────────────►│ gospeed-server │
│   (client)  │      Data (TCP/UDP streams)          │    (server)    │
│             │◄────────────────────────────────────►│                │
└─────────────┘                                      └────────────────┘
```

- **Control channel**: Length-prefixed JSON over TCP for handshake, test negotiation, and results
- **Data channels**: Raw TCP/UDP streams with sequence numbers and timestamps
- **Session correlation**: Client-initiated data connections identified by 16-byte session UUID

## Building from source

```sh
go install github.com/go-task/task/v3/cmd/task@latest

task build              # Build both binaries
task test               # Run tests
task docker             # Build Docker image
task release-snapshot   # Test GoReleaser locally
task bump VERSION=1.2.2 # Tag and push a release
```

<p align="center">
  <img src="docs/assets/gospeed_slogan_banner.png" alt="gospeed — Know Your Network" width="100%">
</p>

## License

MIT

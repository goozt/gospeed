# gospeed

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
gospeed-server -addr :8080

# With TLS
gospeed-server -tls-cert cert.pem -tls-key key.pem
```

### Client

```sh
# Run default test suite against a server
gospeed -s speed.apps.n2h.me

# Use environment variable instead of -s flag
export GOSPEED_SERVER_ADDR=speed.apps.n2h.me:9000
gospeed

# Default: localhost:9000
gospeed
```

### Test selection

```sh
# Run specific tests
gospeed -s host:9000 -latency
gospeed -s host:9000 -tcp -udp
gospeed -s host:9000 -bufferbloat

# Run all tests
gospeed -s host:9000 -all

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
-d  --duration int    Test duration in seconds (default 10)
-h, --history         Show previous results with trends
-v, --version         Print version and exit
    --json            Output as JSON
    --csv             Output as CSV
    --tls             Use TLS connection
    --tls-skip-verify Skip TLS certificate verification

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
┌─────────────┐     Control (TCP, JSON-framed)      ┌──────────────┐
│   gospeed   │◄────────────────────────────────────►│ gospeed-     │
│   (client)  │     Data (TCP/UDP streams)           │   server     │
│             │◄────────────────────────────────────►│              │
└─────────────┘                                      └──────────────┘
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
```

## License

MIT

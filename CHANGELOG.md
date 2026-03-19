# Changelog

All notable changes to this project will be documented in this file.

## [Unreleased]

## [v1.2.2] - 2026-03-19

### Fixed
- Data race in TCP progress loop using channel-based synchronization
- CI test failures on Windows due to PowerShell argument parsing
- Removed real domain from README examples

## [v1.2.1] - 2026-03-19

### Improved
- Comprehensive test coverage across all packages (protocol, server, results, tests)
- Reduced cyclomatic complexity in 10 functions via focused helper extraction
- Code formatting cleanup with gofmt -s
- CI pipeline with coverage reporting via Codecov
- Added project badges (CI, coverage, Go Report Card, Go Reference, License, Release)

## [v1.2.0] - 2026-03-19

### Added
- `--ping` flag for quick server reachability check with 5 retries
- Server advertises supported tests in handshake; client skips unsupported tests

## [v1.1.0] - 2026-03-19

### Added
- TLS support with self-signed certificates and Let's Encrypt ACME
- Shorthand flags (`-s`, `-h`, `-t`, `-d`, `-v`, `-a`) for CLI
- Build info retrieval for version, commit, and date

### Fixed
- `.gitignore` updated to include `node_modules`

## [v1.0.3] - 2026-03-19

### Fixed
- Archive configuration to use `ids` instead of `builds` for correct artifact referencing

## [v1.0.1] - 2026-03-19

### Added
- Windows ARM architecture support in build configurations

## [v1.0.0] - 2026-03-19

### Added
- Core speed testing: latency, TCP/UDP throughput, jitter, bufferbloat, MTU discovery, DNS, TCP connect, bidirectional
- Quality grading system (A-F)
- JSON and CSV output formats
- Test history tracking
- Docker support with UDP

### Changed
- Migrated build tasks from Makefile to Taskfile
- Improved context handling and error reporting
- Simplified context cancellation checks in tests

### Fixed
- Default port appended for server connections

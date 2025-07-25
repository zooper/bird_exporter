# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Prometheus metrics exporter for the BIRD routing daemon, written in Go. The exporter communicates with BIRD via Unix sockets to collect and expose routing protocol metrics.

## Build and Development Commands

```bash
# Build the project
go build

# Run tests with coverage
go test ./... -v -covermode=count

# Run the exporter (requires BIRD daemon running)
./bird_exporter -format.new=true

# Build and run with common flags
go build && ./bird_exporter -bird.socket=/var/run/bird.ctl -format.new=true
```

## Architecture Overview

The codebase follows a layered architecture:

### Core Components

- **main.go**: Entry point with CLI flag parsing and HTTP server setup
- **metric_collector.go**: Central orchestrator that collects metrics from all enabled protocols
- **client/**: BIRD daemon communication layer via Unix sockets
- **metrics/**: Protocol-specific metric exporters with two format strategies (legacy and new)
- **protocol/**: Data structures representing BIRD routing protocols
- **parser/**: Parsers for BIRD daemon output (OSPF, BFD protocols)

### Key Abstractions

- **MetricExporter interface**: All protocol exporters implement this for Prometheus metric collection
- **Client interface**: Abstraction for BIRD daemon communication
- **Protocol struct**: Common data structure for all routing protocol information

### Metric Format Strategies

The exporter supports two metric formats:
- **Legacy format**: Protocol-specific metric names (e.g., `bgp4_session_prefix_count_import`)
- **New format**: Generic format with labels (e.g., `bird_protocol_prefix_import_count{proto="BGP",ip_version="4"}`)

Default is new format (`-format.new=true`), controlled in metric_collector.go:22-26.

### Protocol Support

Supported protocols are defined as bit flags in protocol/protocol.go:3-13:
- BGP, OSPF, Kernel, Static, Direct, Babel, RPKI, BFD

Each protocol has dedicated exporters in metrics/ directory with protocol-specific logic.

## Testing

Tests are located alongside source files with `*_test.go` naming:
- parser/: Tests for BIRD output parsing logic
- metrics/: Tests for label strategies

## BIRD Integration

The exporter requires BIRD routing daemon to be running with accessible Unix socket files:
- Default BIRD socket: `/var/run/bird.ctl`
- Default BIRD6 socket: `/var/run/bird6.ctl` (pre-v2.0)
- BIRD v2.0+ uses single socket for both IPv4/IPv6

## Configuration

All configuration is via CLI flags defined in main.go:17-41. Key flags:
- `-bird.v2`: Enable BIRD v2.0+ mode (single socket for IPv4/IPv6)
- `-bird.socket`: Path to BIRD Unix socket
- `-format.new`: Use new metric format (default: true)
- `-proto.*`: Enable/disable specific protocol metrics
- `-prefix.size`: Enable prefix size statistics collection (default: false)

## Prefix Size Statistics

New feature that collects statistics on route prefix lengths (e.g., /22, /24, etc.):

### Usage
Enable with: `./bird_exporter -prefix.size=true`

### Metrics Exported
- `bird_prefix_length_count{name="bgp1",proto="BGP",ip_version="4",prefix_length="24"}`: Count of prefixes by length

### Implementation Details
- Uses `show route protocol {name}` BIRD command
- Parses route output to extract CIDR prefix lengths
- Supports both IPv4 and IPv6 prefixes
- Works with BGP, OSPF, Kernel, Static, Direct, and Babel protocols
- Parser handles various BIRD route output formats

### Files Added
- `parser/route.go`: Route parsing and prefix length extraction
- `parser/route_test.go`: Tests for route parsing functionality
- `metrics/prefix_size_exporter.go`: Prometheus metrics exporter for prefix statistics
- `protocol/protocol.go`: Added Route and PrefixStats data structures
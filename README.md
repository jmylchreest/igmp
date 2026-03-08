# igmp

[![CI](https://github.com/jmylchreest/igmp/actions/workflows/ci.yml/badge.svg)](https://github.com/jmylchreest/igmp/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/jmylchreest/igmp.svg)](https://pkg.go.dev/github.com/jmylchreest/igmp)

A comprehensive, spec-compliant IGMP (Internet Group Management Protocol) library for Go, with CLI tools for querying and monitoring.

## Features

### Library (`github.com/jmylchreest/igmp`)

- **Full IGMP v1/v2/v3 support** -- marshal and unmarshal all message types
- **Message types**: Membership Query, Reports (v1/v2/v3), Leave Group
- **IGMPv3 Group Records**: all 6 record types (MODE_IS_INCLUDE, MODE_IS_EXCLUDE, CHANGE_TO_INCLUDE, CHANGE_TO_EXCLUDE, ALLOW_NEW_SOURCES, BLOCK_OLD_SOURCES)
- **IGMPv3 floating-point encoding** for Max Response Code and QQIC fields
- **RFC-compliant**: Router Alert IP option, DSCP CS6 TOS, TTL=1
- **Raw socket abstraction** with send and receive support
- **Periodic querier** with context-based lifecycle
- **Passive listener** for monitoring IGMP traffic
- **Minimal dependencies**: only `golang.org/x/net` for the library itself

### CLI Tools

- **`igmpqd`** -- lightweight IGMP query daemon (sends periodic IGMPv2/v3 queries)
- **`igmpmon`** -- real-time IGMP traffic monitor with full TUI dashboard

## Installation

### CLI tools

```bash
go install github.com/jmylchreest/igmp/cmd/igmpqd@latest
go install github.com/jmylchreest/igmp/cmd/igmpmon@latest
```

### Library

```bash
go get github.com/jmylchreest/igmp
```

## Quick Start

### Library Usage

```go
package main

import (
    "fmt"
    "net"
    "time"

    "github.com/jmylchreest/igmp"
)

func main() {
    // Build an IGMPv2 General Query
    query := &igmp.MembershipQuery{
        MaxRespTime:  10 * time.Second,
        GroupAddress: net.IPv4zero,
    }
    data, _ := query.Marshal()
    fmt.Printf("Query: %x\n", data)

    // Parse an incoming IGMP message
    msg, _ := igmp.Parse(data)
    fmt.Printf("Type: %s, Version: %d\n",
        igmp.MessageTypeName(msg.Type()), msg.Version())

    // Build an IGMPv3 Membership Report
    report := &igmp.MembershipReportV3{
        GroupRecords: []igmp.GroupRecord{
            {
                RecordType:   igmp.RecordModeIsExclude,
                GroupAddress: net.IPv4(239, 1, 1, 1),
            },
        },
    }
    data, _ = report.Marshal()
    fmt.Printf("Report: %x\n", data)
}
```

### igmpqd -- Query Daemon

```bash
# Send IGMPv2 General Queries every 30 seconds on eth0
sudo igmpqd run --interface eth0 --interval 30

# Send IGMPv3 queries with debug logging
sudo igmpqd run -I eth0 -V 3 --debug

# Group-specific query
sudo igmpqd run -I eth0 --grp-address 239.1.1.1 --dst-address 239.1.1.1
```

**Flags:**

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--interface` | `-I` | (none) | Network interface to send on |
| `--grp-address` | `-g` | `0.0.0.0` | Group address (0.0.0.0 = General Query) |
| `--dst-address` | `-d` | `224.0.0.1` | Destination IP |
| `--interval` | `-i` | `125` | Seconds between queries |
| `--ttl` | `-t` | `1` | IP TTL |
| `--max-response-time` | `-m` | `100` | Max response time (1/10 sec units) |
| `--version` | `-V` | `2` | IGMP version (2 or 3) |
| `--debug` | | `false` | Enable debug logging |
| `--no-router-alert` | | `false` | Disable IP Router Alert option |

Backward-compatible aliases (`--grpAddress`, `--dstAddress`, `--maxResponseTime`) are also supported.

### igmpmon -- Traffic Monitor

```bash
# Monitor all IGMP traffic on eth0
sudo igmpmon --interface eth0

# Monitor on all interfaces
sudo igmpmon
```

The TUI dashboard shows:
- **Statistics**: packet counts, version breakdown, session duration
- **Querier Status**: detected querier IP, version, last seen
- **Active Groups**: multicast groups with member counts and status
- **Packet Log**: scrollable live feed of all IGMP packets

Press `Enter` on a group to drill into member details (shows individual member IPs, filter mode, sources, and status). Press `Esc` to go back.

**Note on group members:** IGMPv2 uses report suppression, so only one host per group responds per query cycle -- member counts may be understated. IGMPv3 does not suppress reports, so member visibility is much better.

**Keyboard shortcuts:**

| Key | Action |
|-----|--------|
| `Tab` | Switch between Groups and Packet Log panes |
| `Enter` | Drill into selected group's members |
| `Esc` | Back to main view |
| `Up`/`Down` or `k`/`j` | Navigate |
| `f` | Freeze/unfreeze display |
| `q` | Quit |

## Docker

Both tools require `NET_RAW` capability for raw IGMP sockets.

```bash
# Build
docker build -t igmp .

# Run querier daemon
docker run --cap-add=NET_RAW --network=host igmp run --interface eth0

# Run monitor (requires TTY)
docker run -it --cap-add=NET_RAW --network=host igmp /usr/local/bin/igmpmon -I eth0
```

## Specifications

This library implements:

| Specification | Description |
|---------------|-------------|
| [RFC 1112](https://www.rfc-editor.org/rfc/rfc1112) | IGMPv1 -- Host Extensions for IP Multicasting |
| [RFC 2236](https://www.rfc-editor.org/rfc/rfc2236) | IGMPv2 -- Internet Group Management Protocol, Version 2 |
| [RFC 3376](https://www.rfc-editor.org/rfc/rfc3376) | IGMPv3 -- Internet Group Management Protocol, Version 3 |
| [RFC 2113](https://www.rfc-editor.org/rfc/rfc2113) | IP Router Alert Option |
| [RFC 1071](https://www.rfc-editor.org/rfc/rfc1071) | Internet Checksum |

## Requirements

- Go 1.25+ (built with Go 1.26)
- Linux or macOS (raw sockets)
- Root privileges or `CAP_NET_RAW` capability

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Run tests: `go test -race ./...`
4. Commit your changes
5. Push to the branch
6. Open a Pull Request

## License

[MIT](LICENSE)

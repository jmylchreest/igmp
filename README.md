# igmpqd

IGMPQD (IGMP Query Daemon) is a lightweight utility to periodically generate IGMPv2 Query messages. It is designed to be simple and do just enough to maintain IGMP memberships in environments that may not have a dedicated IGMP Querier.

The message payload can be configured as needed, within RFC specifications (see [RFC 2236, Section 2](https://tools.ietf.org/html/rfc2236#section-2)).

If this utility is useful to you, great! If you find a need to extend its functionality or fix a bug, even better – please fork the repository and submit a pull request.

## Usage

### Pre-compiled Binaries

Pre-compiled binaries for various platforms and architectures are typically provided in `.zip` files. These can be found in the `dist/` directory of a release (e.g., `dist/linux_amd64.zip`). Download the appropriate binary for your system, extract it, and you're ready to go.

### Building from Source

Alternatively, you can build `igmpqd` from source.

#### Prerequisites
*   **Go**: Version 1.23 or higher.
*   **Make**: (Optional, for using the Makefile)
*   Standard build tools (a C compiler for Go, etc.)

#### Build Steps
1.  **Clone the repository**:
    ```bash
    git clone https://github.com/jmylchreest/igmpqd.git
    cd igmpqd
    ```
2.  **Build using Makefile (recommended for release binaries)**:
    The provided `Makefile` handles cross-compilation for multiple platforms and architectures, placing the packaged binaries in the `./dist` directory.
    ```bash
    make all
    ```
    This will create binaries like `dist/linux_amd64/igmpqd` and then package them into zip files like `dist/linux_amd64.zip`.

3.  **Build using standard Go commands (for local development)**:
    For a quick local build, you can use the standard `go build` command. This will create a binary for your current system in the project root.
    ```bash
    go build .
    ```

### Print Version

Executing `./igmpqd version` (or `igmpqd.exe version` on Windows) will display version information:

```
$ ./igmpqd version
Version:	0.0.2-1-g2ce0283 (Commit: 2ce0283ece778c8d9e2cd1adfa84453534c05e46)
Built:		Fri, 30 May 2025 14:17:23 +0000
Fingerprint:	gc/linux/amd64/go1.23.9
```
*(Note: The actual version, commit, build time, and Go version will vary depending on the build.)*

### Run the Daemon

`igmpqd` does not fork or background itself. It is designed to be managed by a process supervisor like systemd, supervisor, or similar tools.

To run the daemon, use the `run` command: `./igmpqd run`. The `run` command accepts several flags to customize its behavior:

```
Usage:
  igmpqd run [flags]

Flags:
      --debug                 Enable debug messages to stderr. (default false)
  -d, --dstAddress string     Specified IP address to send the IGMP Query to. (default "224.0.0.1")
  -g, --grpAddress string     Specified IP address to use as the Group Address. Used to query for specific group members. (default "0.0.0.0")
  -I, --interface string      Specified network interface to send the IGMP Query. (no default)
  -i, --interval int          The time in seconds to delay between sending IGMP Query messages. (default 30)
  -m, --maxResponseTime int   Specifies the maximum allowed time before sending a responding report in units of 1/10 second. (default 100)
  -t, --ttl int               The TTL of the IGMP Query. (default 1)
```

## Development Details

*   **Go Version**: The project uses Go 1.23, as specified in the `go.mod` file.
*   **Dependencies**: Dependencies are managed using Go Modules. The `go.mod` file lists all direct and indirect dependencies.
*   **Linting & Formatting**: Code is formatted using `gofmt`. `staticcheck` is used for linting.
*   **Testing**: Basic unit tests are included and can be run with `go test ./...`.

## Contributing

Contributions are welcome! Please follow these general steps:
1.  Fork the repository.
2.  Create a new branch for your feature or bug fix.
3.  Make your changes. Ensure code is formatted (`gofmt`) and passes linting (`staticcheck`).
4.  Add or update tests for your changes.
5.  Submit a pull request with a clear description of your changes.
```

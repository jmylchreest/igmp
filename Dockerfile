# SPDX-License-Identifier: MIT

# Build stage
FROM golang:1.26 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w \
    -X github.com/jmylchreest/igmp/internal/version.GitCommit=$(git rev-parse --short HEAD 2>/dev/null || echo unknown) \
    -X github.com/jmylchreest/igmp/internal/version.GitDescribe=$(git describe --tags 2>/dev/null || echo dev)" \
    -o /igmpqd ./cmd/igmpqd

RUN CGO_ENABLED=0 go build -ldflags="-s -w \
    -X github.com/jmylchreest/igmp/internal/version.GitCommit=$(git rev-parse --short HEAD 2>/dev/null || echo unknown) \
    -X github.com/jmylchreest/igmp/internal/version.GitDescribe=$(git describe --tags 2>/dev/null || echo dev)" \
    -o /igmpmon ./cmd/igmpmon

# Runtime stage — scratch for minimal attack surface.
# NOTE: This container requires NET_RAW capability to send/receive IGMP packets.
#
# Run with:
#   docker run --cap-add=NET_RAW --network=host igmpqd run --interface eth0
FROM scratch
COPY --from=build /igmpqd /usr/local/bin/igmpqd
COPY --from=build /igmpmon /usr/local/bin/igmpmon
ENTRYPOINT ["igmpqd"]

// SPDX-License-Identifier: MIT

package igmp

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"time"
)

// QuerierConfig configures the periodic IGMP query sender.
type QuerierConfig struct {
	// Interface is the network interface to send queries on.
	// If empty, queries are sent on all interfaces.
	Interface string

	// Interval is the time between General Queries.
	// Default: 125 seconds (RFC 3376 Section 8.2).
	Interval time.Duration

	// TTL is the IP TTL for query packets. Default: 1.
	TTL int

	// GroupAddress is the multicast group to query.
	// Default: 0.0.0.0 (General Query — queries all groups).
	GroupAddress net.IP

	// DestAddress is the destination IP for query packets.
	// Default: 224.0.0.1 (All Hosts).
	DestAddress net.IP

	// MaxResponseTime is the maximum time hosts have to respond.
	// Default: 10 seconds (100 in 1/10s units per RFC 2236).
	MaxResponseTime time.Duration

	// Version is the IGMP version to use (2 or 3). Default: 2.
	Version int

	// Logger is an optional structured logger. If nil, a no-op logger is used.
	Logger *slog.Logger

	// RouterAlert controls whether the IP Router Alert option is included.
	// Default: true. Use [Bool] to set explicitly.
	RouterAlert *bool

	// IGMPv3-only fields.

	// RobustnessVariable is the Querier's Robustness Variable (QRV).
	// Default: 2 (RFC 3376 Section 8.1).
	RobustnessVariable uint8

	// SourceAddresses limits queries to specific sources (IGMPv3 only).
	SourceAddresses []net.IP

	// SuppressRouter sets the S flag in IGMPv3 queries.
	SuppressRouter bool
}

// Querier sends periodic IGMP membership queries.
//
// It wraps a [Conn] and a ticker to send queries at a configured interval.
// Errors during sending are logged but do not stop the querier — this is
// important for long-running daemons where transient network errors should
// not be fatal.
type Querier struct {
	config QuerierConfig
	conn   *Conn
	logger *slog.Logger
}

// NewQuerier creates a new IGMP querier with the given configuration.
//
// The querier does not start sending until [Querier.Run] is called.
func NewQuerier(cfg QuerierConfig) (*Querier, error) {
	// Apply defaults.
	if cfg.Interval <= 0 {
		cfg.Interval = time.Duration(DefaultQueryInterval) * time.Second
	}
	if cfg.TTL <= 0 {
		cfg.TTL = DefaultTTL
	}
	if cfg.GroupAddress == nil {
		cfg.GroupAddress = net.IPv4zero
	}
	if cfg.DestAddress == nil {
		cfg.DestAddress = AllHosts
	}
	if cfg.MaxResponseTime <= 0 {
		cfg.MaxResponseTime = time.Duration(DefaultMaxResponseTime) * 100 * time.Millisecond
	}
	if cfg.Version == 0 {
		cfg.Version = 2
	}
	if cfg.RobustnessVariable == 0 && cfg.Version == 3 {
		cfg.RobustnessVariable = DefaultRobustnessVariable
	}
	if cfg.RouterAlert == nil {
		cfg.RouterAlert = Bool(true)
	}

	logger := cfg.Logger
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}

	if cfg.Version != 2 && cfg.Version != 3 {
		return nil, fmt.Errorf("igmp: unsupported query version %d (must be 2 or 3)", cfg.Version)
	}

	conn, err := NewConn(ConnConfig{
		Interface:   cfg.Interface,
		TTL:         cfg.TTL,
		TOS:         DefaultTOS,
		RouterAlert: cfg.RouterAlert,
	})
	if err != nil {
		return nil, fmt.Errorf("igmp: create connection: %w", err)
	}

	return &Querier{
		config: cfg,
		conn:   conn,
		logger: logger,
	}, nil
}

// Run starts the querier and blocks until ctx is cancelled.
//
// It sends a query immediately on start, then every [QuerierConfig.Interval].
// Send errors are logged but do not stop the querier.
func (q *Querier) Run(ctx context.Context) error {
	q.logger.Info("querier started",
		"version", q.config.Version,
		"interface", q.config.Interface,
		"interval", q.config.Interval,
		"group", q.config.GroupAddress,
		"destination", q.config.DestAddress,
		"ttl", q.config.TTL,
		"max_response_time", q.config.MaxResponseTime,
	)

	// Send initial query.
	q.sendQuery()

	ticker := time.NewTicker(q.config.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			q.logger.Info("querier stopping")
			q.conn.Close()
			return ctx.Err()
		case <-ticker.C:
			q.sendQuery()
		}
	}
}

func (q *Querier) sendQuery() {
	query := q.buildQuery()

	if err := q.conn.Send(q.config.DestAddress, query); err != nil {
		q.logger.Error("failed to send IGMP query",
			"error", err,
			"destination", q.config.DestAddress,
			"group", q.config.GroupAddress,
		)
		return
	}

	q.logger.Debug("sent IGMP query",
		"version", query.Version(),
		"destination", q.config.DestAddress,
		"group", q.config.GroupAddress,
	)
}

func (q *Querier) buildQuery() *MembershipQuery {
	query := &MembershipQuery{
		MaxRespTime:  q.config.MaxResponseTime,
		GroupAddress: q.config.GroupAddress,
	}

	if q.config.Version == 3 {
		query.SuppressRouter = q.config.SuppressRouter
		query.QRV = q.config.RobustnessVariable
		query.QQIC = q.config.Interval
		query.SourceAddresses = q.config.SourceAddresses
	}

	return query
}

// Close stops the querier and releases resources.
func (q *Querier) Close() error {
	return q.conn.Close()
}

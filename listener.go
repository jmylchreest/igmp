// SPDX-License-Identifier: MIT

package igmp

import (
	"context"
	"fmt"
	"log/slog"
	"net"
)

// ListenerConfig configures the IGMP listener.
type ListenerConfig struct {
	// Interface is the network interface to listen on.
	// If empty, listens on all interfaces.
	Interface string

	// Logger is an optional structured logger. If nil, a no-op logger is used.
	Logger *slog.Logger

	// BufferSize is the channel buffer size for received messages.
	// Default: 256.
	BufferSize int

	// JoinAllRouters controls whether the listener also joins the
	// All Routers multicast group (224.0.0.2) in addition to All Hosts
	// (224.0.0.1). This is useful for seeing Leave Group messages, which
	// are sent to All Routers. Default: false.
	JoinAllRouters bool

	// Groups is an optional list of additional multicast groups to join.
	// The listener always joins All Hosts (224.0.0.1); specify extra
	// groups here to also receive their traffic.
	Groups []net.IP
}

// Listener receives IGMP messages from the network and delivers them
// as parsed [ReceivedMessage] values on a channel.
//
// It operates as a passive observer — it does not send any traffic.
// This is useful for monitoring and diagnostics.
//
// On creation, the listener automatically joins the All Hosts multicast
// group (224.0.0.1) and optionally All Routers (224.0.0.2) to ensure
// IGMP traffic is delivered to the socket. Additional groups can be
// specified via [ListenerConfig.Groups].
type Listener struct {
	config       ListenerConfig
	conn         *Conn
	logger       *slog.Logger
	joinedGroups []net.IP
}

// NewListener creates a new IGMP listener.
//
// Opening a raw socket requires elevated privileges (root or CAP_NET_RAW).
// The listener joins the All Hosts group (224.0.0.1) on creation. Set
// [ListenerConfig.JoinAllRouters] to also join All Routers (224.0.0.2).
func NewListener(cfg ListenerConfig) (*Listener, error) {
	if cfg.BufferSize <= 0 {
		cfg.BufferSize = 256
	}

	logger := cfg.Logger
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}

	conn, err := NewConn(ConnConfig{
		Interface: cfg.Interface,
	})
	if err != nil {
		return nil, fmt.Errorf("igmp: create listener connection: %w", err)
	}

	l := &Listener{
		config: cfg,
		conn:   conn,
		logger: logger,
	}

	// Join multicast groups so the kernel delivers IGMP traffic to this socket.
	groups := []net.IP{AllHosts}
	if cfg.JoinAllRouters {
		groups = append(groups, AllRouters)
	}
	groups = append(groups, cfg.Groups...)

	for _, g := range groups {
		if err := conn.JoinGroup(g); err != nil {
			// Log but don't fail — some groups may already be joined or the
			// OS may implicitly join AllHosts.
			logger.Warn("failed to join multicast group", "group", g, "error", err)
			continue
		}
		l.joinedGroups = append(l.joinedGroups, g)
		logger.Debug("joined multicast group", "group", g)
	}

	return l, nil
}

// Listen starts receiving IGMP messages and delivers them on the returned channel.
//
// The channel is closed when the context is cancelled or an unrecoverable error
// occurs. Malformed packets and unknown message types are silently skipped.
//
// Listen blocks until the context is cancelled.
func (l *Listener) Listen(ctx context.Context) (<-chan *ReceivedMessage, error) {
	ch := make(chan *ReceivedMessage, l.config.BufferSize)

	l.logger.Info("listener started",
		"interface", l.config.Interface,
		"buffer_size", l.config.BufferSize,
	)

	go func() {
		defer close(ch)
		defer l.conn.Close()

		for {
			msg, err := l.conn.Receive(ctx)
			if err != nil {
				if ctx.Err() != nil {
					l.logger.Info("listener stopping")
					return
				}
				l.logger.Error("listener receive error", "error", err)
				return
			}

			l.logger.Debug("received IGMP message",
				"type", MessageTypeName(msg.Message.Type()),
				"version", msg.Message.Version(),
				"source", msg.Source,
			)

			select {
			case ch <- msg:
			case <-ctx.Done():
				return
			default:
				// Channel full, drop oldest behavior would require a ring buffer.
				// For now, we log and skip.
				l.logger.Warn("listener channel full, dropping message")
			}
		}
	}()

	return ch, nil
}

// Close leaves all joined multicast groups and closes the underlying connection.
func (l *Listener) Close() error {
	for _, g := range l.joinedGroups {
		if err := l.conn.LeaveGroup(g); err != nil {
			l.logger.Warn("failed to leave multicast group", "group", g, "error", err)
		}
	}
	l.joinedGroups = nil
	return l.conn.Close()
}

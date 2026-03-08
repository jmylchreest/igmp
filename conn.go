// SPDX-License-Identifier: MIT

package igmp

import (
	"context"
	"fmt"
	"net"
	"runtime"
	"sync"
	"time"

	"golang.org/x/net/ipv4"
)

// ConnConfig configures a raw IGMP connection.
type ConnConfig struct {
	// Interface is the network interface name to bind to (e.g., "eth0").
	// If empty, the connection is not bound to a specific interface.
	Interface string

	// TTL is the IP Time-To-Live for sent packets. Default: 1 (per RFC 2236).
	TTL int

	// TOS is the IP Type-Of-Service byte. Default: 0xc0 (DSCP CS6).
	TOS int

	// RouterAlert controls whether the IP Router Alert option (RFC 2113)
	// is included in sent packets. Default: true (per RFC 2236 Section 2).
	// Use [Bool] to set explicitly.
	RouterAlert *bool
}

// Bool returns a pointer to b. This is a convenience helper for setting
// optional boolean fields like [ConnConfig.RouterAlert].
func Bool(b bool) *bool { return &b }

// Conn wraps a raw IPv4 socket for sending and receiving IGMP messages.
//
// It handles IP header construction, the Router Alert option, TOS, and TTL
// as required by the IGMP specifications.
type Conn struct {
	rawConn    *ipv4.RawConn
	packetConn net.PacketConn
	config     ConnConfig
	iface      *net.Interface
	mu         sync.Mutex
	closed     bool
}

// NewConn creates a new raw IGMP connection.
//
// Opening a raw socket requires elevated privileges (root or CAP_NET_RAW).
func NewConn(cfg ConnConfig) (*Conn, error) {
	// Apply defaults.
	if cfg.TTL <= 0 {
		cfg.TTL = DefaultTTL
	}
	if cfg.TOS == 0 {
		cfg.TOS = DefaultTOS
	}
	// Default RouterAlert to true if not explicitly set.
	if cfg.RouterAlert == nil {
		cfg.RouterAlert = Bool(true)
	}

	c, err := net.ListenPacket("ip4:2", "0.0.0.0") // IGMP = IP protocol 2
	if err != nil {
		return nil, fmt.Errorf("igmp: failed to open raw socket: %w", err)
	}

	r, err := ipv4.NewRawConn(c)
	if err != nil {
		c.Close()
		return nil, fmt.Errorf("igmp: failed to create raw connection: %w", err)
	}

	conn := &Conn{
		rawConn:    r,
		packetConn: c,
		config:     cfg,
	}

	if cfg.Interface != "" {
		iface, err := net.InterfaceByName(cfg.Interface)
		if err != nil {
			c.Close()
			return nil, fmt.Errorf("igmp: interface %q: %w", cfg.Interface, err)
		}
		conn.iface = iface
	}

	return conn, nil
}

// Send sends an IGMP message to the specified destination IP address.
//
// The IP header is constructed automatically with the configured TTL, TOS,
// and Router Alert option.
func (c *Conn) Send(dst net.IP, msg Message) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return fmt.Errorf("igmp: connection is closed")
	}

	dstV4 := dst.To4()
	if dstV4 == nil {
		return fmt.Errorf("%w: destination %v is not IPv4", ErrInvalidIP, dst)
	}

	payload, err := msg.Marshal()
	if err != nil {
		return fmt.Errorf("igmp: marshal: %w", err)
	}

	iph := &ipv4.Header{
		Version:  ipv4.Version,
		Len:      ipv4.HeaderLen,
		TOS:      c.config.TOS,
		TotalLen: ipv4.HeaderLen + len(payload),
		TTL:      c.config.TTL,
		Protocol: ProtocolNumber,
		Dst:      dstV4,
	}

	// Add Router Alert option (RFC 2113) if configured.
	// Router Alert is a 4-byte IP option: 0x94, 0x04, 0x00, 0x00
	if c.config.RouterAlert != nil && *c.config.RouterAlert {
		iph.Options = []byte{0x94, 0x04, 0x00, 0x00}
		iph.Len = ipv4.HeaderLen + 4
		iph.TotalLen = iph.Len + len(payload)
	}

	var cm *ipv4.ControlMessage
	if c.iface != nil {
		switch runtime.GOOS {
		case "darwin", "linux":
			cm = &ipv4.ControlMessage{IfIndex: c.iface.Index}
		default:
			if err := c.rawConn.SetMulticastInterface(c.iface); err != nil {
				return fmt.Errorf("igmp: set multicast interface: %w", err)
			}
		}
	}

	if err := c.rawConn.WriteTo(iph, payload, cm); err != nil {
		return fmt.Errorf("igmp: write: %w", err)
	}

	return nil
}

// ReceivedMessage wraps a parsed IGMP Message with metadata from the IP layer.
type ReceivedMessage struct {
	// Message is the parsed IGMP message.
	Message Message

	// Source is the source IP address from the IP header.
	Source net.IP

	// Received is the time the message was received.
	Received time.Time

	// IfIndex is the interface index the message was received on (if available).
	IfIndex int

	// RawBytes contains the original IGMP payload bytes.
	RawBytes []byte
}

// Receive reads and parses the next IGMP message from the connection.
//
// It blocks until a message is received, the context is cancelled, or an error
// occurs. Unknown IGMP message types are silently skipped.
func (c *Conn) Receive(ctx context.Context) (*ReceivedMessage, error) {
	for {
		// Check context before blocking on read.
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Set a read deadline so we can check context periodically.
		if err := c.packetConn.SetReadDeadline(time.Now().Add(500 * time.Millisecond)); err != nil {
			return nil, fmt.Errorf("igmp: set read deadline: %w", err)
		}

		buf := make([]byte, 1500) // Standard MTU size
		hdr, payload, cm, err := c.rawConn.ReadFrom(buf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue // Timeout, check context and retry.
			}
			if c.closed {
				return nil, fmt.Errorf("igmp: connection closed")
			}
			return nil, fmt.Errorf("igmp: read: %w", err)
		}

		if len(payload) < 8 {
			continue // Too short, skip.
		}

		msg, err := Parse(payload)
		if err != nil {
			continue // Unknown or malformed, skip.
		}

		rm := &ReceivedMessage{
			Message:  msg,
			Received: time.Now(),
			RawBytes: make([]byte, len(payload)),
		}
		copy(rm.RawBytes, payload)

		if hdr != nil {
			rm.Source = hdr.Src
		}
		if cm != nil {
			rm.IfIndex = cm.IfIndex
		}

		return rm, nil
	}
}

// JoinGroup joins a multicast group on the connection's interface.
//
// This is required to receive IGMP traffic destined for a specific multicast
// group address. The kernel will begin delivering packets for this group
// to the socket.
//
// If no interface was configured, the system default multicast interface is used.
func (c *Conn) JoinGroup(group net.IP) error {
	groupV4 := group.To4()
	if groupV4 == nil {
		return fmt.Errorf("%w: %v is not IPv4", ErrInvalidIP, group)
	}
	if err := c.rawConn.JoinGroup(c.iface, &net.IPAddr{IP: groupV4}); err != nil {
		return fmt.Errorf("igmp: join group %v: %w", groupV4, err)
	}
	return nil
}

// LeaveGroup leaves a previously joined multicast group.
//
// After leaving, the kernel stops delivering packets for this group to the socket.
func (c *Conn) LeaveGroup(group net.IP) error {
	groupV4 := group.To4()
	if groupV4 == nil {
		return fmt.Errorf("%w: %v is not IPv4", ErrInvalidIP, group)
	}
	if err := c.rawConn.LeaveGroup(c.iface, &net.IPAddr{IP: groupV4}); err != nil {
		return fmt.Errorf("igmp: leave group %v: %w", groupV4, err)
	}
	return nil
}

// JoinSourceSpecificGroup joins a source-specific multicast group (IGMPv3 SSM).
//
// This subscribes to traffic for the given multicast group only from the
// specified source address. This maps to an IGMPv3 INCLUDE mode subscription.
func (c *Conn) JoinSourceSpecificGroup(group, source net.IP) error {
	groupV4 := group.To4()
	if groupV4 == nil {
		return fmt.Errorf("%w: group %v is not IPv4", ErrInvalidIP, group)
	}
	sourceV4 := source.To4()
	if sourceV4 == nil {
		return fmt.Errorf("%w: source %v is not IPv4", ErrInvalidIP, source)
	}
	if err := c.rawConn.JoinSourceSpecificGroup(c.iface, &net.IPAddr{IP: groupV4}, &net.IPAddr{IP: sourceV4}); err != nil {
		return fmt.Errorf("igmp: join SSM group %v source %v: %w", groupV4, sourceV4, err)
	}
	return nil
}

// LeaveSourceSpecificGroup leaves a source-specific multicast group.
func (c *Conn) LeaveSourceSpecificGroup(group, source net.IP) error {
	groupV4 := group.To4()
	if groupV4 == nil {
		return fmt.Errorf("%w: group %v is not IPv4", ErrInvalidIP, group)
	}
	sourceV4 := source.To4()
	if sourceV4 == nil {
		return fmt.Errorf("%w: source %v is not IPv4", ErrInvalidIP, source)
	}
	if err := c.rawConn.LeaveSourceSpecificGroup(c.iface, &net.IPAddr{IP: groupV4}, &net.IPAddr{IP: sourceV4}); err != nil {
		return fmt.Errorf("igmp: leave SSM group %v source %v: %w", groupV4, sourceV4, err)
	}
	return nil
}

// IncludeSourceSpecificGroup adds a source to an existing source-specific
// multicast group subscription.
//
// This is used to expand the set of sources for an existing SSM subscription
// without leaving and rejoining the group.
func (c *Conn) IncludeSourceSpecificGroup(group, source net.IP) error {
	groupV4 := group.To4()
	if groupV4 == nil {
		return fmt.Errorf("%w: group %v is not IPv4", ErrInvalidIP, group)
	}
	sourceV4 := source.To4()
	if sourceV4 == nil {
		return fmt.Errorf("%w: source %v is not IPv4", ErrInvalidIP, source)
	}
	if err := c.rawConn.IncludeSourceSpecificGroup(c.iface, &net.IPAddr{IP: groupV4}, &net.IPAddr{IP: sourceV4}); err != nil {
		return fmt.Errorf("igmp: include SSM source %v for group %v: %w", sourceV4, groupV4, err)
	}
	return nil
}

// ExcludeSourceSpecificGroup removes a source from an existing source-specific
// multicast group subscription.
func (c *Conn) ExcludeSourceSpecificGroup(group, source net.IP) error {
	groupV4 := group.To4()
	if groupV4 == nil {
		return fmt.Errorf("%w: group %v is not IPv4", ErrInvalidIP, group)
	}
	sourceV4 := source.To4()
	if sourceV4 == nil {
		return fmt.Errorf("%w: source %v is not IPv4", ErrInvalidIP, source)
	}
	if err := c.rawConn.ExcludeSourceSpecificGroup(c.iface, &net.IPAddr{IP: groupV4}, &net.IPAddr{IP: sourceV4}); err != nil {
		return fmt.Errorf("igmp: exclude SSM source %v for group %v: %w", sourceV4, groupV4, err)
	}
	return nil
}

// SetMulticastLoopback controls whether multicast packets sent on this
// connection are looped back to the sending host.
//
// This is useful when a host runs both a querier and a listener: set to true
// so the listener sees the querier's own queries. Default is system-dependent
// (usually true).
func (c *Conn) SetMulticastLoopback(on bool) error {
	if err := c.rawConn.SetMulticastLoopback(on); err != nil {
		return fmt.Errorf("igmp: set multicast loopback: %w", err)
	}
	return nil
}

// MulticastLoopback reports whether multicast loopback is enabled.
func (c *Conn) MulticastLoopback() (bool, error) {
	on, err := c.rawConn.MulticastLoopback()
	if err != nil {
		return false, fmt.Errorf("igmp: get multicast loopback: %w", err)
	}
	return on, nil
}

// SetMulticastInterface sets the network interface for outgoing multicast
// packets. This overrides the interface set at construction time.
func (c *Conn) SetMulticastInterface(ifi *net.Interface) error {
	if err := c.rawConn.SetMulticastInterface(ifi); err != nil {
		return fmt.Errorf("igmp: set multicast interface: %w", err)
	}
	c.mu.Lock()
	c.iface = ifi
	c.mu.Unlock()
	return nil
}

// Interface returns the network interface this connection is bound to,
// or nil if no specific interface was configured.
func (c *Conn) Interface() *net.Interface {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.iface
}

// Close closes the underlying raw socket.
func (c *Conn) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}
	c.closed = true
	return c.packetConn.Close()
}

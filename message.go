// SPDX-License-Identifier: MIT

package igmp

import (
	"errors"
)

// Message is the interface implemented by all IGMP message types.
type Message interface {
	// Type returns the IGMP message type code (e.g., TypeMembershipQuery).
	Type() uint8

	// Version returns the IGMP protocol version this message belongs to (1, 2, or 3).
	Version() int

	// Marshal serializes the IGMP message into wire format bytes.
	// The checksum field is computed automatically.
	Marshal() ([]byte, error)
}

// Common errors returned by Parse and Marshal operations.
var (
	ErrTooShort      = errors.New("igmp: message too short")
	ErrInvalidType   = errors.New("igmp: unknown message type")
	ErrBadChecksum   = errors.New("igmp: invalid checksum")
	ErrInvalidIP     = errors.New("igmp: invalid IPv4 address")
	ErrInvalidLength = errors.New("igmp: inconsistent message length")
)

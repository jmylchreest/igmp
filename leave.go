// SPDX-License-Identifier: MIT

package igmp

import (
	"encoding/binary"
	"fmt"
	"net"
)

// LeaveGroup represents an IGMPv2 Leave Group message (type 0x17).
//
// A host sends a Leave Group message to the all-routers multicast
// address (224.0.0.2) when it leaves a multicast group and it was
// the last host to reply to a query with a Membership Report.
//
// Wire format (8 bytes):
//
//	+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//	|  Type = 0x17  |   Max Resp    |           Checksum            |
//	+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//	|                         Group Address                         |
//	+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
type LeaveGroup struct {
	// GroupAddress is the multicast group being left.
	GroupAddress net.IP
}

// Type returns the IGMP message type code.
func (l *LeaveGroup) Type() uint8 { return TypeLeaveGroupV2 }

// Version returns 2.
func (l *LeaveGroup) Version() int { return 2 }

// Marshal serializes the Leave Group message to wire format.
func (l *LeaveGroup) Marshal() ([]byte, error) {
	grp, err := requireIPv4(l.GroupAddress, "group address")
	if err != nil {
		return nil, err
	}

	buf := make([]byte, 8)
	buf[0] = TypeLeaveGroupV2
	// buf[1] = 0 (unused in Leave messages)
	copy(buf[4:8], grp)
	binary.BigEndian.PutUint16(buf[2:4], Checksum(buf, 0))
	return buf, nil
}

// ParseLeaveGroup decodes an IGMPv2 Leave Group message from wire format.
func ParseLeaveGroup(data []byte) (*LeaveGroup, error) {
	if len(data) < 8 {
		return nil, fmt.Errorf("%w: need 8 bytes, got %d", ErrTooShort, len(data))
	}
	if data[0] != TypeLeaveGroupV2 {
		return nil, fmt.Errorf("%w: expected 0x17, got 0x%02x", ErrInvalidType, data[0])
	}
	if !ValidateChecksum(data[:8]) {
		return nil, ErrBadChecksum
	}

	l := &LeaveGroup{
		GroupAddress: net.IP(make([]byte, 4)),
	}
	copy(l.GroupAddress, data[4:8])
	return l, nil
}

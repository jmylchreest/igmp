// SPDX-License-Identifier: MIT

package igmp

import (
	"encoding/binary"
	"fmt"
	"net"
	"time"
)

// MembershipQuery represents an IGMP Membership Query message (type 0x11).
//
// A single type covers all three IGMP versions:
//   - IGMPv1: 8 bytes, MaxRespCode = 0
//   - IGMPv2: 8 bytes, MaxRespCode > 0
//   - IGMPv3: 12+ bytes, includes S flag, QRV, QQIC, source list
//
// Wire format (IGMPv2, 8 bytes):
//
//	 0                   1                   2                   3
//	 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
//	+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//	|  Type = 0x11  | Max Resp Code |           Checksum            |
//	+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//	|                         Group Address                         |
//	+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//
// Wire format (IGMPv3, 12+ bytes):
//
//	+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//	|  Type = 0x11  | Max Resp Code |           Checksum            |
//	+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//	|                         Group Address                         |
//	+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//	| Resv  |S| QRV |     QQIC      |     Number of Sources (N)     |
//	+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//	|                       Source Address [1]                       |
//	+-                                                             -+
//	|                              ...                              |
type MembershipQuery struct {
	// MaxRespTime is the maximum time allowed before sending a responding
	// report. For IGMPv2 this is encoded linearly in 1/10 second units (0-255).
	// For IGMPv3, values >= 128 use a floating-point encoding.
	MaxRespTime time.Duration

	// GroupAddress is the multicast group being queried.
	// 0.0.0.0 for a General Query; a specific group for Group-Specific
	// or Group-and-Source-Specific queries.
	GroupAddress net.IP

	// IGMPv3 fields. These are zero-valued for v1/v2 queries.

	// SuppressRouter is the S flag (Suppress Router-Side Processing).
	SuppressRouter bool

	// QRV is the Querier's Robustness Variable.
	QRV uint8

	// QQIC is the Querier's Query Interval, decoded from the QQIC code.
	// Uses the same floating-point encoding as MaxRespCode for values >= 128.
	QQIC time.Duration

	// SourceAddresses is the list of source addresses for
	// Group-and-Source-Specific Queries (IGMPv3 only).
	SourceAddresses []net.IP
}

// Type returns the IGMP message type code.
func (q *MembershipQuery) Type() uint8 {
	return TypeMembershipQuery
}

// Version returns the IGMP version inferred from the message fields.
// A v3 query is identified by having QRV, QQIC, or SourceAddresses set,
// or by a Marshal call that produces > 8 bytes.
// A v1 query has MaxRespTime == 0.
// Otherwise it's v2.
func (q *MembershipQuery) Version() int {
	if q.isV3() {
		return 3
	}
	if q.MaxRespTime == 0 {
		return 1
	}
	return 2
}

func (q *MembershipQuery) isV3() bool {
	return q.QRV > 0 || q.QQIC > 0 || len(q.SourceAddresses) > 0 || q.SuppressRouter
}

// IsGeneral returns true if this is a General Query (group 0.0.0.0, no sources).
func (q *MembershipQuery) IsGeneral() bool {
	return (q.GroupAddress == nil || q.GroupAddress.Equal(net.IPv4zero)) &&
		len(q.SourceAddresses) == 0
}

// IsGroupSpecific returns true if this is a Group-Specific Query.
func (q *MembershipQuery) IsGroupSpecific() bool {
	return q.GroupAddress != nil && !q.GroupAddress.Equal(net.IPv4zero) &&
		len(q.SourceAddresses) == 0
}

// IsGroupAndSourceSpecific returns true if this is a Group-and-Source-Specific Query.
func (q *MembershipQuery) IsGroupAndSourceSpecific() bool {
	return q.GroupAddress != nil && !q.GroupAddress.Equal(net.IPv4zero) &&
		len(q.SourceAddresses) > 0
}

// Marshal serializes the MembershipQuery to wire format.
// The checksum is computed automatically.
func (q *MembershipQuery) Marshal() ([]byte, error) {
	grp := net.IPv4zero.To4()
	if q.GroupAddress != nil {
		grp = q.GroupAddress.To4()
		if grp == nil {
			return nil, fmt.Errorf("%w: group address is not IPv4", ErrInvalidIP)
		}
	}

	for _, src := range q.SourceAddresses {
		if src.To4() == nil {
			return nil, fmt.Errorf("%w: source address %v is not IPv4", ErrInvalidIP, src)
		}
	}

	if q.isV3() {
		return q.marshalV3(grp)
	}
	return q.marshalV1V2(grp)
}

func (q *MembershipQuery) marshalV1V2(grp net.IP) ([]byte, error) {
	buf := make([]byte, 8)
	buf[0] = TypeMembershipQuery
	buf[1] = encodeMaxRespCode(q.MaxRespTime, false)
	// Checksum at bytes 2-3, set to 0 for computation.
	copy(buf[4:8], grp)

	binary.BigEndian.PutUint16(buf[2:4], Checksum(buf, 0))
	return buf, nil
}

func (q *MembershipQuery) marshalV3(grp net.IP) ([]byte, error) {
	numSources := len(q.SourceAddresses)
	length := 12 + numSources*4
	buf := make([]byte, length)

	buf[0] = TypeMembershipQuery
	buf[1] = encodeMaxRespCode(q.MaxRespTime, true)
	// Checksum at bytes 2-3, set to 0 for computation.
	copy(buf[4:8], grp)

	// Byte 8: Resv (4 bits) | S (1 bit) | QRV (3 bits)
	var flags uint8
	if q.SuppressRouter {
		flags |= 0x08
	}
	flags |= q.QRV & 0x07
	buf[8] = flags

	// Byte 9: QQIC
	buf[9] = encodeQQIC(q.QQIC)

	// Bytes 10-11: Number of Sources
	binary.BigEndian.PutUint16(buf[10:12], uint16(numSources))

	// Source addresses
	for i, src := range q.SourceAddresses {
		copy(buf[12+i*4:16+i*4], src.To4())
	}

	binary.BigEndian.PutUint16(buf[2:4], Checksum(buf, 0))
	return buf, nil
}

// ParseMembershipQuery decodes a Membership Query from wire format.
// The data should start at the IGMP type byte (not the IP header).
func ParseMembershipQuery(data []byte) (*MembershipQuery, error) {
	if len(data) < 8 {
		return nil, fmt.Errorf("%w: need at least 8 bytes, got %d", ErrTooShort, len(data))
	}

	if data[0] != TypeMembershipQuery {
		return nil, fmt.Errorf("%w: expected 0x11, got 0x%02x", ErrInvalidType, data[0])
	}

	if !ValidateChecksum(data) {
		return nil, ErrBadChecksum
	}

	q := &MembershipQuery{
		GroupAddress: net.IP(make([]byte, 4)),
	}
	copy(q.GroupAddress, data[4:8])

	if len(data) == 8 {
		// IGMPv1 or IGMPv2 query.
		q.MaxRespTime = decodeMaxRespCode(data[1], false)
		return q, nil
	}

	// IGMPv3 query (12+ bytes).
	if len(data) < 12 {
		return nil, fmt.Errorf("%w: v3 query needs at least 12 bytes, got %d", ErrTooShort, len(data))
	}

	q.MaxRespTime = decodeMaxRespCode(data[1], true)
	q.SuppressRouter = (data[8] & 0x08) != 0
	q.QRV = data[8] & 0x07
	q.QQIC = decodeQQIC(data[9])

	numSources := int(binary.BigEndian.Uint16(data[10:12]))
	expectedLen := 12 + numSources*4
	if len(data) < expectedLen {
		return nil, fmt.Errorf("%w: v3 query declares %d sources but only %d bytes available",
			ErrInvalidLength, numSources, len(data))
	}

	q.SourceAddresses = make([]net.IP, numSources)
	for i := range numSources {
		q.SourceAddresses[i] = net.IP(make([]byte, 4))
		copy(q.SourceAddresses[i], data[12+i*4:16+i*4])
	}

	return q, nil
}

// encodeMaxRespCode encodes a duration into the Max Resp Code field.
//
// For IGMPv2 (v3ok=false): linear encoding in 1/10 second units, capped at 255.
//
// For IGMPv3 (v3ok=true): values < 128 are encoded linearly.
// Values >= 128 use a floating-point encoding:
//
//	0 1 2 3 4 5 6 7
//	+-+-+-+-+-+-+-+-+
//	|1| exp | mant  |
//	+-+-+-+-+-+-+-+-+
//
// value = (mant | 0x10) << (exp + 3)
func encodeMaxRespCode(d time.Duration, v3ok bool) uint8 {
	// Convert to 1/10 second units.
	tenths := int(d / (100 * time.Millisecond))
	if tenths < 0 {
		tenths = 0
	}

	if tenths < 128 || !v3ok {
		if tenths > 255 {
			tenths = 255
		}
		return uint8(tenths)
	}

	// IGMPv3 floating-point encoding.
	return encodeFloatingPoint(tenths)
}

// decodeMaxRespCode decodes the Max Resp Code field to a duration.
func decodeMaxRespCode(code uint8, v3 bool) time.Duration {
	var tenths int
	if code < 128 || !v3 {
		tenths = int(code)
	} else {
		tenths = decodeFloatingPoint(code)
	}
	return time.Duration(tenths) * 100 * time.Millisecond
}

// encodeQQIC encodes a duration into the QQIC field.
// Uses the same floating-point encoding as MaxRespCode but in seconds.
func encodeQQIC(d time.Duration) uint8 {
	secs := int(d / time.Second)
	if secs < 0 {
		secs = 0
	}
	if secs < 128 {
		return uint8(secs)
	}
	return encodeFloatingPoint(secs)
}

// decodeQQIC decodes the QQIC field to a duration.
func decodeQQIC(code uint8) time.Duration {
	var secs int
	if code < 128 {
		secs = int(code)
	} else {
		secs = decodeFloatingPoint(code)
	}
	return time.Duration(secs) * time.Second
}

// encodeFloatingPoint encodes a value >= 128 into the IGMPv3 floating-point
// format: 1 | exp(3) | mant(4), where value = (mant | 0x10) << (exp + 3).
func encodeFloatingPoint(value int) uint8 {
	if value < 128 {
		return uint8(value)
	}

	// Find the highest representable value <= the input.
	for exp := 7; exp >= 0; exp-- {
		for mant := 15; mant >= 0; mant-- {
			v := (mant | 0x10) << (uint(exp) + 3)
			if v <= value {
				return 0x80 | uint8(exp<<4) | uint8(mant)
			}
		}
	}
	// Maximum representable value.
	return 0xff
}

// decodeFloatingPoint decodes the IGMPv3 floating-point format.
func decodeFloatingPoint(code uint8) int {
	mant := int(code & 0x0f)
	exp := int((code >> 4) & 0x07)
	return (mant | 0x10) << (uint(exp) + 3)
}

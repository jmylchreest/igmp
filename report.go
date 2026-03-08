// SPDX-License-Identifier: MIT

package igmp

import (
	"encoding/binary"
	"fmt"
	"net"
)

// MembershipReportV1 represents an IGMPv1 Membership Report (type 0x12).
//
// Wire format (8 bytes):
//
//	+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//	|  Type = 0x12  |   Unused      |           Checksum            |
//	+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//	|                         Group Address                         |
//	+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
type MembershipReportV1 struct {
	// GroupAddress is the multicast group being reported.
	GroupAddress net.IP
}

// Type returns the IGMP message type code.
func (r *MembershipReportV1) Type() uint8 { return TypeMembershipReportV1 }

// Version returns 1.
func (r *MembershipReportV1) Version() int { return 1 }

// Marshal serializes the report to wire format.
func (r *MembershipReportV1) Marshal() ([]byte, error) {
	grp, err := requireIPv4(r.GroupAddress, "group address")
	if err != nil {
		return nil, err
	}

	buf := make([]byte, 8)
	buf[0] = TypeMembershipReportV1
	// buf[1] = 0 (unused)
	copy(buf[4:8], grp)
	binary.BigEndian.PutUint16(buf[2:4], Checksum(buf, 0))
	return buf, nil
}

// ParseMembershipReportV1 decodes an IGMPv1 Membership Report from wire format.
func ParseMembershipReportV1(data []byte) (*MembershipReportV1, error) {
	if len(data) < 8 {
		return nil, fmt.Errorf("%w: need 8 bytes, got %d", ErrTooShort, len(data))
	}
	if data[0] != TypeMembershipReportV1 {
		return nil, fmt.Errorf("%w: expected 0x12, got 0x%02x", ErrInvalidType, data[0])
	}
	if !ValidateChecksum(data[:8]) {
		return nil, ErrBadChecksum
	}

	r := &MembershipReportV1{
		GroupAddress: net.IP(make([]byte, 4)),
	}
	copy(r.GroupAddress, data[4:8])
	return r, nil
}

// MembershipReportV2 represents an IGMPv2 Membership Report (type 0x16).
//
// Wire format (8 bytes):
//
//	+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//	|  Type = 0x16  | Max Resp Time |           Checksum            |
//	+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//	|                         Group Address                         |
//	+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
type MembershipReportV2 struct {
	// GroupAddress is the multicast group being reported.
	GroupAddress net.IP
}

// Type returns the IGMP message type code.
func (r *MembershipReportV2) Type() uint8 { return TypeMembershipReportV2 }

// Version returns 2.
func (r *MembershipReportV2) Version() int { return 2 }

// Marshal serializes the report to wire format.
func (r *MembershipReportV2) Marshal() ([]byte, error) {
	grp, err := requireIPv4(r.GroupAddress, "group address")
	if err != nil {
		return nil, err
	}

	buf := make([]byte, 8)
	buf[0] = TypeMembershipReportV2
	// buf[1] = 0 (max resp time, unused in reports)
	copy(buf[4:8], grp)
	binary.BigEndian.PutUint16(buf[2:4], Checksum(buf, 0))
	return buf, nil
}

// ParseMembershipReportV2 decodes an IGMPv2 Membership Report from wire format.
func ParseMembershipReportV2(data []byte) (*MembershipReportV2, error) {
	if len(data) < 8 {
		return nil, fmt.Errorf("%w: need 8 bytes, got %d", ErrTooShort, len(data))
	}
	if data[0] != TypeMembershipReportV2 {
		return nil, fmt.Errorf("%w: expected 0x16, got 0x%02x", ErrInvalidType, data[0])
	}
	if !ValidateChecksum(data[:8]) {
		return nil, ErrBadChecksum
	}

	r := &MembershipReportV2{
		GroupAddress: net.IP(make([]byte, 4)),
	}
	copy(r.GroupAddress, data[4:8])
	return r, nil
}

// GroupRecord is a single record within an IGMPv3 Membership Report.
//
// Wire format:
//
//	+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//	|  Record Type  |  Aux Data Len |     Number of Sources (N)     |
//	+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//	|                       Multicast Address                       |
//	+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//	|                       Source Address [1]                       |
//	+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//	|                              ...                              |
//	+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//	|                       Auxiliary Data                           |
//	+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
type GroupRecord struct {
	// RecordType indicates the type of this record (e.g., RecordModeIsInclude).
	RecordType uint8

	// GroupAddress is the multicast group address for this record.
	GroupAddress net.IP

	// SourceAddresses is the list of source addresses in this record.
	SourceAddresses []net.IP

	// AuxData contains auxiliary data. Per RFC 3376, this is rarely used
	// and implementations should ignore it if not understood.
	AuxData []byte
}

// RecordName returns the human-readable name for this record's type.
func (gr *GroupRecord) RecordName() string {
	return RecordTypeName(gr.RecordType)
}

// marshal serializes a single GroupRecord.
func (gr *GroupRecord) marshal() ([]byte, error) {
	grp, err := requireIPv4(gr.GroupAddress, "group record address")
	if err != nil {
		return nil, err
	}

	for _, src := range gr.SourceAddresses {
		if src.To4() == nil {
			return nil, fmt.Errorf("%w: source address %v is not IPv4", ErrInvalidIP, src)
		}
	}

	numSources := len(gr.SourceAddresses)
	auxDataLen := len(gr.AuxData)
	// Aux Data Len is in 32-bit words.
	auxWords := (auxDataLen + 3) / 4

	recordLen := 8 + numSources*4 + auxWords*4
	buf := make([]byte, recordLen)

	buf[0] = gr.RecordType
	buf[1] = uint8(auxWords)
	binary.BigEndian.PutUint16(buf[2:4], uint16(numSources))
	copy(buf[4:8], grp)

	for i, src := range gr.SourceAddresses {
		copy(buf[8+i*4:12+i*4], src.To4())
	}

	if auxDataLen > 0 {
		copy(buf[8+numSources*4:], gr.AuxData)
	}

	return buf, nil
}

// MembershipReportV3 represents an IGMPv3 Membership Report (type 0x22).
//
// Wire format:
//
//	+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//	|  Type = 0x22  |    Reserved   |           Checksum            |
//	+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//	|           Reserved            |  Number of Group Records (M)  |
//	+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//	|                         Group Record [1]                      |
//	+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//	|                              ...                              |
type MembershipReportV3 struct {
	// GroupRecords contains the group records in this report.
	GroupRecords []GroupRecord
}

// Type returns the IGMP message type code.
func (r *MembershipReportV3) Type() uint8 { return TypeMembershipReportV3 }

// Version returns 3.
func (r *MembershipReportV3) Version() int { return 3 }

// Marshal serializes the IGMPv3 report to wire format.
func (r *MembershipReportV3) Marshal() ([]byte, error) {
	// Fixed header: 8 bytes
	header := make([]byte, 8)
	header[0] = TypeMembershipReportV3
	// header[1] = 0 (reserved)
	// header[2:4] = checksum (set later)
	// header[4:6] = 0 (reserved)
	binary.BigEndian.PutUint16(header[6:8], uint16(len(r.GroupRecords)))

	var records []byte
	for i := range r.GroupRecords {
		rec, err := r.GroupRecords[i].marshal()
		if err != nil {
			return nil, fmt.Errorf("group record %d: %w", i, err)
		}
		records = append(records, rec...)
	}

	buf := append(header, records...)
	binary.BigEndian.PutUint16(buf[2:4], Checksum(buf, 0))
	return buf, nil
}

// ParseMembershipReportV3 decodes an IGMPv3 Membership Report from wire format.
func ParseMembershipReportV3(data []byte) (*MembershipReportV3, error) {
	if len(data) < 8 {
		return nil, fmt.Errorf("%w: need at least 8 bytes, got %d", ErrTooShort, len(data))
	}
	if data[0] != TypeMembershipReportV3 {
		return nil, fmt.Errorf("%w: expected 0x22, got 0x%02x", ErrInvalidType, data[0])
	}
	if !ValidateChecksum(data) {
		return nil, ErrBadChecksum
	}

	numRecords := int(binary.BigEndian.Uint16(data[6:8]))
	r := &MembershipReportV3{
		GroupRecords: make([]GroupRecord, 0, numRecords),
	}

	offset := 8
	for range numRecords {
		if offset+8 > len(data) {
			return nil, fmt.Errorf("%w: truncated group record at offset %d", ErrTooShort, offset)
		}

		recordType := data[offset]
		auxDataWords := int(data[offset+1])
		numSources := int(binary.BigEndian.Uint16(data[offset+2 : offset+4]))

		recordLen := 8 + numSources*4 + auxDataWords*4
		if offset+recordLen > len(data) {
			return nil, fmt.Errorf("%w: truncated group record (need %d bytes at offset %d, have %d)",
				ErrInvalidLength, recordLen, offset, len(data)-offset)
		}

		gr := GroupRecord{
			RecordType:   recordType,
			GroupAddress: net.IP(make([]byte, 4)),
		}
		copy(gr.GroupAddress, data[offset+4:offset+8])

		gr.SourceAddresses = make([]net.IP, numSources)
		for i := range numSources {
			gr.SourceAddresses[i] = net.IP(make([]byte, 4))
			copy(gr.SourceAddresses[i], data[offset+8+i*4:offset+12+i*4])
		}

		if auxDataWords > 0 {
			auxStart := offset + 8 + numSources*4
			auxLen := auxDataWords * 4
			gr.AuxData = make([]byte, auxLen)
			copy(gr.AuxData, data[auxStart:auxStart+auxLen])
		}

		r.GroupRecords = append(r.GroupRecords, gr)
		offset += recordLen
	}

	return r, nil
}

// requireIPv4 validates that an IP address is a valid IPv4 address
// and returns its 4-byte representation.
func requireIPv4(ip net.IP, name string) (net.IP, error) {
	if ip == nil {
		return nil, fmt.Errorf("%w: %s is nil", ErrInvalidIP, name)
	}
	v4 := ip.To4()
	if v4 == nil {
		return nil, fmt.Errorf("%w: %s %v is not IPv4", ErrInvalidIP, name, ip)
	}
	return v4, nil
}

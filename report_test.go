// SPDX-License-Identifier: MIT

package igmp

import (
	"net"
	"testing"
)

func TestMembershipReportV1_MarshalRoundTrip(t *testing.T) {
	original := &MembershipReportV1{
		GroupAddress: net.IPv4(239, 1, 1, 1),
	}

	data, err := original.Marshal()
	if err != nil {
		t.Fatalf("Marshal() error: %v", err)
	}

	if len(data) != 8 {
		t.Fatalf("Marshal() length = %d, want 8", len(data))
	}
	if data[0] != TypeMembershipReportV1 {
		t.Errorf("type = 0x%02x, want 0x%02x", data[0], TypeMembershipReportV1)
	}

	parsed, err := ParseMembershipReportV1(data)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	if parsed.Version() != 1 {
		t.Errorf("Version() = %d, want 1", parsed.Version())
	}
	if parsed.Type() != TypeMembershipReportV1 {
		t.Errorf("Type() = 0x%02x, want 0x%02x", parsed.Type(), TypeMembershipReportV1)
	}
	if !parsed.GroupAddress.Equal(original.GroupAddress.To4()) {
		t.Errorf("GroupAddress = %v, want %v", parsed.GroupAddress, original.GroupAddress)
	}
}

func TestMembershipReportV2_MarshalRoundTrip(t *testing.T) {
	original := &MembershipReportV2{
		GroupAddress: net.IPv4(239, 255, 255, 250),
	}

	data, err := original.Marshal()
	if err != nil {
		t.Fatalf("Marshal() error: %v", err)
	}

	if len(data) != 8 {
		t.Fatalf("Marshal() length = %d, want 8", len(data))
	}
	if data[0] != TypeMembershipReportV2 {
		t.Errorf("type = 0x%02x, want 0x%02x", data[0], TypeMembershipReportV2)
	}

	parsed, err := ParseMembershipReportV2(data)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	if parsed.Version() != 2 {
		t.Errorf("Version() = %d, want 2", parsed.Version())
	}
	if !parsed.GroupAddress.Equal(original.GroupAddress.To4()) {
		t.Errorf("GroupAddress = %v, want %v", parsed.GroupAddress, original.GroupAddress)
	}
}

func TestMembershipReportV3_MarshalRoundTrip(t *testing.T) {
	original := &MembershipReportV3{
		GroupRecords: []GroupRecord{
			{
				RecordType:   RecordModeIsExclude,
				GroupAddress: net.IPv4(239, 1, 1, 1),
				SourceAddresses: []net.IP{
					net.IPv4(10, 0, 0, 1),
					net.IPv4(10, 0, 0, 2),
				},
			},
			{
				RecordType:      RecordModeIsInclude,
				GroupAddress:    net.IPv4(239, 2, 2, 2),
				SourceAddresses: []net.IP{},
			},
		},
	}

	data, err := original.Marshal()
	if err != nil {
		t.Fatalf("Marshal() error: %v", err)
	}

	if data[0] != TypeMembershipReportV3 {
		t.Errorf("type = 0x%02x, want 0x%02x", data[0], TypeMembershipReportV3)
	}

	parsed, err := ParseMembershipReportV3(data)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	if parsed.Version() != 3 {
		t.Errorf("Version() = %d, want 3", parsed.Version())
	}
	if len(parsed.GroupRecords) != 2 {
		t.Fatalf("len(GroupRecords) = %d, want 2", len(parsed.GroupRecords))
	}

	// First record.
	gr0 := parsed.GroupRecords[0]
	if gr0.RecordType != RecordModeIsExclude {
		t.Errorf("GroupRecords[0].RecordType = %d, want %d", gr0.RecordType, RecordModeIsExclude)
	}
	if !gr0.GroupAddress.Equal(net.IPv4(239, 1, 1, 1).To4()) {
		t.Errorf("GroupRecords[0].GroupAddress = %v, want 239.1.1.1", gr0.GroupAddress)
	}
	if len(gr0.SourceAddresses) != 2 {
		t.Fatalf("GroupRecords[0] sources = %d, want 2", len(gr0.SourceAddresses))
	}
	if !gr0.SourceAddresses[0].Equal(net.IPv4(10, 0, 0, 1)) {
		t.Errorf("source[0] = %v, want 10.0.0.1", gr0.SourceAddresses[0])
	}
	if !gr0.SourceAddresses[1].Equal(net.IPv4(10, 0, 0, 2)) {
		t.Errorf("source[1] = %v, want 10.0.0.2", gr0.SourceAddresses[1])
	}

	// Second record.
	gr1 := parsed.GroupRecords[1]
	if gr1.RecordType != RecordModeIsInclude {
		t.Errorf("GroupRecords[1].RecordType = %d, want %d", gr1.RecordType, RecordModeIsInclude)
	}
	if !gr1.GroupAddress.Equal(net.IPv4(239, 2, 2, 2).To4()) {
		t.Errorf("GroupRecords[1].GroupAddress = %v, want 239.2.2.2", gr1.GroupAddress)
	}
	if len(gr1.SourceAddresses) != 0 {
		t.Errorf("GroupRecords[1] sources = %d, want 0", len(gr1.SourceAddresses))
	}
}

func TestMembershipReportV3_WithAuxData(t *testing.T) {
	original := &MembershipReportV3{
		GroupRecords: []GroupRecord{
			{
				RecordType:   RecordModeIsExclude,
				GroupAddress: net.IPv4(239, 1, 1, 1),
				AuxData:      []byte{0x01, 0x02, 0x03, 0x04},
			},
		},
	}

	data, err := original.Marshal()
	if err != nil {
		t.Fatalf("Marshal() error: %v", err)
	}

	parsed, err := ParseMembershipReportV3(data)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	if len(parsed.GroupRecords[0].AuxData) != 4 {
		t.Errorf("AuxData length = %d, want 4", len(parsed.GroupRecords[0].AuxData))
	}
}

func TestMembershipReportV3_Empty(t *testing.T) {
	original := &MembershipReportV3{
		GroupRecords: []GroupRecord{},
	}

	data, err := original.Marshal()
	if err != nil {
		t.Fatalf("Marshal() error: %v", err)
	}

	if len(data) != 8 {
		t.Fatalf("empty report length = %d, want 8", len(data))
	}

	parsed, err := ParseMembershipReportV3(data)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	if len(parsed.GroupRecords) != 0 {
		t.Errorf("len(GroupRecords) = %d, want 0", len(parsed.GroupRecords))
	}
}

func TestMembershipReportV3_AllRecordTypes(t *testing.T) {
	recordTypes := []uint8{
		RecordModeIsInclude,
		RecordModeIsExclude,
		RecordChangeToInclude,
		RecordChangeToExclude,
		RecordAllowNewSources,
		RecordBlockOldSources,
	}

	for _, rt := range recordTypes {
		t.Run(RecordTypeName(rt), func(t *testing.T) {
			original := &MembershipReportV3{
				GroupRecords: []GroupRecord{
					{
						RecordType:      rt,
						GroupAddress:    net.IPv4(239, 1, 1, 1),
						SourceAddresses: []net.IP{net.IPv4(10, 0, 0, 1)},
					},
				},
			}

			data, err := original.Marshal()
			if err != nil {
				t.Fatalf("Marshal() error: %v", err)
			}

			parsed, err := ParseMembershipReportV3(data)
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			if parsed.GroupRecords[0].RecordType != rt {
				t.Errorf("RecordType = %d, want %d", parsed.GroupRecords[0].RecordType, rt)
			}
		})
	}
}

func TestMembershipReport_InvalidIP(t *testing.T) {
	tests := []struct {
		name string
		msg  Message
	}{
		{
			name: "v1 nil",
			msg:  &MembershipReportV1{GroupAddress: nil},
		},
		{
			name: "v2 IPv6",
			msg:  &MembershipReportV2{GroupAddress: net.ParseIP("::1")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.msg.Marshal()
			if err == nil {
				t.Error("expected error for invalid IP")
			}
		})
	}
}

func TestMembershipReportV3_Truncated(t *testing.T) {
	// Build a valid report, then truncate it.
	original := &MembershipReportV3{
		GroupRecords: []GroupRecord{
			{
				RecordType:      RecordModeIsExclude,
				GroupAddress:    net.IPv4(239, 1, 1, 1),
				SourceAddresses: []net.IP{net.IPv4(10, 0, 0, 1)},
			},
		},
	}

	data, err := original.Marshal()
	if err != nil {
		t.Fatalf("Marshal() error: %v", err)
	}

	// Truncate the source address.
	_, err = ParseMembershipReportV3(data[:12])
	if err == nil {
		t.Error("expected error for truncated data")
	}
}

func TestGroupRecord_RecordName(t *testing.T) {
	tests := []struct {
		rt   uint8
		name string
	}{
		{RecordModeIsInclude, "MODE_IS_INCLUDE"},
		{RecordModeIsExclude, "MODE_IS_EXCLUDE"},
		{RecordChangeToInclude, "CHANGE_TO_INCLUDE_MODE"},
		{RecordChangeToExclude, "CHANGE_TO_EXCLUDE_MODE"},
		{RecordAllowNewSources, "ALLOW_NEW_SOURCES"},
		{RecordBlockOldSources, "BLOCK_OLD_SOURCES"},
		{99, "UNKNOWN"},
	}

	for _, tt := range tests {
		gr := &GroupRecord{RecordType: tt.rt}
		if got := gr.RecordName(); got != tt.name {
			t.Errorf("RecordName(%d) = %q, want %q", tt.rt, got, tt.name)
		}
	}
}

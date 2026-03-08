// SPDX-License-Identifier: MIT

package igmp

import (
	"net"
	"testing"
	"time"
)

func TestParse_Dispatch(t *testing.T) {
	// Build valid messages and verify Parse dispatches correctly.
	tests := []struct {
		name        string
		msg         Message
		expectedVer int
		expectedTyp uint8
	}{
		{
			name:        "v1 query",
			msg:         &MembershipQuery{GroupAddress: net.IPv4zero},
			expectedVer: 1,
			expectedTyp: TypeMembershipQuery,
		},
		{
			name:        "v2 query",
			msg:         &MembershipQuery{MaxRespTime: 10 * time.Second, GroupAddress: net.IPv4zero},
			expectedVer: 2,
			expectedTyp: TypeMembershipQuery,
		},
		{
			name:        "v1 report",
			msg:         &MembershipReportV1{GroupAddress: net.IPv4(239, 1, 1, 1)},
			expectedVer: 1,
			expectedTyp: TypeMembershipReportV1,
		},
		{
			name:        "v2 report",
			msg:         &MembershipReportV2{GroupAddress: net.IPv4(239, 1, 1, 1)},
			expectedVer: 2,
			expectedTyp: TypeMembershipReportV2,
		},
		{
			name:        "v2 leave",
			msg:         &LeaveGroup{GroupAddress: net.IPv4(239, 1, 1, 1)},
			expectedVer: 2,
			expectedTyp: TypeLeaveGroupV2,
		},
		{
			name: "v3 report",
			msg: &MembershipReportV3{
				GroupRecords: []GroupRecord{
					{
						RecordType:   RecordModeIsExclude,
						GroupAddress: net.IPv4(239, 1, 1, 1),
					},
				},
			},
			expectedVer: 3,
			expectedTyp: TypeMembershipReportV3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := tt.msg.Marshal()
			if err != nil {
				t.Fatalf("Marshal() error: %v", err)
			}

			parsed, err := Parse(data)
			if err != nil {
				t.Fatalf("Parse() error: %v", err)
			}

			if parsed.Type() != tt.expectedTyp {
				t.Errorf("Type() = 0x%02x, want 0x%02x", parsed.Type(), tt.expectedTyp)
			}
			if parsed.Version() != tt.expectedVer {
				t.Errorf("Version() = %d, want %d", parsed.Version(), tt.expectedVer)
			}
		})
	}
}

func TestParse_TooShort(t *testing.T) {
	_, err := Parse([]byte{0x11})
	if err == nil {
		t.Error("expected error for short data")
	}
}

func TestParse_UnknownType(t *testing.T) {
	data := []byte{0x99, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	_, err := Parse(data)
	if err == nil {
		t.Error("expected error for unknown type")
	}
}

func TestMessageTypeName(t *testing.T) {
	tests := []struct {
		typ  uint8
		name string
	}{
		{TypeMembershipQuery, "Membership Query"},
		{TypeMembershipReportV1, "IGMPv1 Membership Report"},
		{TypeMembershipReportV2, "IGMPv2 Membership Report"},
		{TypeLeaveGroupV2, "IGMPv2 Leave Group"},
		{TypeMembershipReportV3, "IGMPv3 Membership Report"},
		{0x99, "Unknown"},
	}

	for _, tt := range tests {
		if got := MessageTypeName(tt.typ); got != tt.name {
			t.Errorf("MessageTypeName(0x%02x) = %q, want %q", tt.typ, got, tt.name)
		}
	}
}

// SPDX-License-Identifier: MIT

package igmp

import (
	"net"
	"testing"
	"time"
)

func TestMembershipQuery_Version(t *testing.T) {
	tests := []struct {
		name    string
		query   MembershipQuery
		version int
	}{
		{
			name:    "v1 query (MaxRespTime zero)",
			query:   MembershipQuery{},
			version: 1,
		},
		{
			name:    "v2 query (MaxRespTime non-zero)",
			query:   MembershipQuery{MaxRespTime: 10 * time.Second},
			version: 2,
		},
		{
			name:    "v3 query (QRV set)",
			query:   MembershipQuery{MaxRespTime: 10 * time.Second, QRV: 2},
			version: 3,
		},
		{
			name:    "v3 query (sources set)",
			query:   MembershipQuery{SourceAddresses: []net.IP{net.IPv4(10, 0, 0, 1)}},
			version: 3,
		},
		{
			name:    "v3 query (suppress router)",
			query:   MembershipQuery{SuppressRouter: true},
			version: 3,
		},
		{
			name:    "v3 query (QQIC set)",
			query:   MembershipQuery{QQIC: 125 * time.Second},
			version: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.query.Version(); got != tt.version {
				t.Errorf("Version() = %d, want %d", got, tt.version)
			}
		})
	}
}

func TestMembershipQuery_QueryType(t *testing.T) {
	tests := []struct {
		name                   string
		query                  MembershipQuery
		general, groupSpecific bool
		groupAndSource         bool
	}{
		{
			name:    "general query (nil group)",
			query:   MembershipQuery{},
			general: true,
		},
		{
			name:    "general query (zero group)",
			query:   MembershipQuery{GroupAddress: net.IPv4zero},
			general: true,
		},
		{
			name:          "group-specific query",
			query:         MembershipQuery{GroupAddress: net.IPv4(239, 1, 1, 1)},
			groupSpecific: true,
		},
		{
			name: "group-and-source-specific query",
			query: MembershipQuery{
				GroupAddress:    net.IPv4(239, 1, 1, 1),
				SourceAddresses: []net.IP{net.IPv4(10, 0, 0, 1)},
			},
			groupAndSource: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.query.IsGeneral(); got != tt.general {
				t.Errorf("IsGeneral() = %v, want %v", got, tt.general)
			}
			if got := tt.query.IsGroupSpecific(); got != tt.groupSpecific {
				t.Errorf("IsGroupSpecific() = %v, want %v", got, tt.groupSpecific)
			}
			if got := tt.query.IsGroupAndSourceSpecific(); got != tt.groupAndSource {
				t.Errorf("IsGroupAndSourceSpecific() = %v, want %v", got, tt.groupAndSource)
			}
		})
	}
}

func TestMembershipQuery_V2_MarshalRoundTrip(t *testing.T) {
	original := &MembershipQuery{
		MaxRespTime:  10 * time.Second, // 100 tenths
		GroupAddress: net.IPv4zero,
	}

	data, err := original.Marshal()
	if err != nil {
		t.Fatalf("Marshal() error: %v", err)
	}

	if len(data) != 8 {
		t.Fatalf("Marshal() length = %d, want 8", len(data))
	}

	if data[0] != TypeMembershipQuery {
		t.Errorf("type byte = 0x%02x, want 0x%02x", data[0], TypeMembershipQuery)
	}
	if data[1] != 100 {
		t.Errorf("max resp code = %d, want 100", data[1])
	}

	parsed, err := ParseMembershipQuery(data)
	if err != nil {
		t.Fatalf("ParseMembershipQuery() error: %v", err)
	}

	if parsed.Version() != 2 {
		t.Errorf("Version() = %d, want 2", parsed.Version())
	}
	if parsed.MaxRespTime != original.MaxRespTime {
		t.Errorf("MaxRespTime = %v, want %v", parsed.MaxRespTime, original.MaxRespTime)
	}
	if !parsed.GroupAddress.Equal(net.IPv4zero) {
		t.Errorf("GroupAddress = %v, want %v", parsed.GroupAddress, net.IPv4zero)
	}
}

func TestMembershipQuery_V1_MarshalRoundTrip(t *testing.T) {
	original := &MembershipQuery{
		MaxRespTime:  0,
		GroupAddress: net.IPv4zero,
	}

	data, err := original.Marshal()
	if err != nil {
		t.Fatalf("Marshal() error: %v", err)
	}

	if data[1] != 0 {
		t.Errorf("max resp code = %d, want 0 for v1", data[1])
	}

	parsed, err := ParseMembershipQuery(data)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	if parsed.Version() != 1 {
		t.Errorf("Version() = %d, want 1", parsed.Version())
	}
}

func TestMembershipQuery_V3_MarshalRoundTrip(t *testing.T) {
	original := &MembershipQuery{
		MaxRespTime:    10 * time.Second,
		GroupAddress:   net.IPv4(239, 1, 1, 1),
		SuppressRouter: true,
		QRV:            3,
		QQIC:           60 * time.Second,
		SourceAddresses: []net.IP{
			net.IPv4(10, 0, 0, 1),
			net.IPv4(10, 0, 0, 2),
		},
	}

	data, err := original.Marshal()
	if err != nil {
		t.Fatalf("Marshal() error: %v", err)
	}

	expectedLen := 12 + 2*4 // 12 header + 2 sources * 4 bytes
	if len(data) != expectedLen {
		t.Fatalf("Marshal() length = %d, want %d", len(data), expectedLen)
	}

	parsed, err := ParseMembershipQuery(data)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	if parsed.Version() != 3 {
		t.Errorf("Version() = %d, want 3", parsed.Version())
	}
	if parsed.MaxRespTime != original.MaxRespTime {
		t.Errorf("MaxRespTime = %v, want %v", parsed.MaxRespTime, original.MaxRespTime)
	}
	if !parsed.GroupAddress.Equal(original.GroupAddress.To4()) {
		t.Errorf("GroupAddress = %v, want %v", parsed.GroupAddress, original.GroupAddress)
	}
	if parsed.SuppressRouter != true {
		t.Error("SuppressRouter = false, want true")
	}
	if parsed.QRV != 3 {
		t.Errorf("QRV = %d, want 3", parsed.QRV)
	}
	if parsed.QQIC != 60*time.Second {
		t.Errorf("QQIC = %v, want %v", parsed.QQIC, 60*time.Second)
	}
	if len(parsed.SourceAddresses) != 2 {
		t.Fatalf("len(SourceAddresses) = %d, want 2", len(parsed.SourceAddresses))
	}
	if !parsed.SourceAddresses[0].Equal(net.IPv4(10, 0, 0, 1)) {
		t.Errorf("SourceAddresses[0] = %v, want 10.0.0.1", parsed.SourceAddresses[0])
	}
	if !parsed.SourceAddresses[1].Equal(net.IPv4(10, 0, 0, 2)) {
		t.Errorf("SourceAddresses[1] = %v, want 10.0.0.2", parsed.SourceAddresses[1])
	}
}

func TestMembershipQuery_V2_GroupSpecific(t *testing.T) {
	original := &MembershipQuery{
		MaxRespTime:  time.Second, // 10 tenths
		GroupAddress: net.IPv4(239, 255, 255, 250),
	}

	data, err := original.Marshal()
	if err != nil {
		t.Fatalf("Marshal() error: %v", err)
	}

	parsed, err := ParseMembershipQuery(data)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	if !parsed.IsGroupSpecific() {
		t.Error("expected IsGroupSpecific() = true")
	}
	if parsed.IsGeneral() {
		t.Error("expected IsGeneral() = false")
	}
}

func TestMembershipQuery_InvalidIP(t *testing.T) {
	q := &MembershipQuery{
		GroupAddress: net.ParseIP("::1"), // IPv6
	}
	_, err := q.Marshal()
	if err == nil {
		t.Error("expected error for IPv6 group address")
	}
}

func TestMembershipQuery_TooShort(t *testing.T) {
	_, err := ParseMembershipQuery([]byte{0x11, 0x00})
	if err == nil {
		t.Error("expected error for short data")
	}
}

func TestMembershipQuery_BadChecksum(t *testing.T) {
	data := []byte{0x11, 0x64, 0xff, 0xff, 0x00, 0x00, 0x00, 0x00}
	_, err := ParseMembershipQuery(data)
	if err == nil {
		t.Error("expected checksum error")
	}
}

func TestFloatingPointEncoding(t *testing.T) {
	tests := []struct {
		name    string
		value   int
		decoded int
	}{
		{"min linear", 0, 0},
		{"max linear", 127, 127},
		{"min float", 128, 128},
		{"large value", 31744, 31744},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded := encodeFloatingPoint(tt.value)
			decoded := tt.value
			if tt.value >= 128 {
				decoded = decodeFloatingPoint(encoded)
			} else {
				decoded = int(encoded)
			}
			if decoded != tt.decoded {
				t.Errorf("encode(%d) = 0x%02x, decode = %d, want %d",
					tt.value, encoded, decoded, tt.decoded)
			}
		})
	}
}

func TestMaxRespCodeRoundTrip(t *testing.T) {
	// Test linear range (0-127 tenths = 0-12.7 seconds).
	for tenths := 0; tenths < 128; tenths++ {
		d := time.Duration(tenths) * 100 * time.Millisecond
		encoded := encodeMaxRespCode(d, true)
		decoded := decodeMaxRespCode(encoded, true)
		if decoded != d {
			t.Errorf("MaxRespCode round-trip failed for %v: encoded=0x%02x, decoded=%v",
				d, encoded, decoded)
		}
	}
}

func TestQQICRoundTrip(t *testing.T) {
	// Test linear range (0-127 seconds).
	for secs := 0; secs < 128; secs++ {
		d := time.Duration(secs) * time.Second
		encoded := encodeQQIC(d)
		decoded := decodeQQIC(encoded)
		if decoded != d {
			t.Errorf("QQIC round-trip failed for %v: encoded=0x%02x, decoded=%v",
				d, encoded, decoded)
		}
	}
}

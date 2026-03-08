// SPDX-License-Identifier: MIT

package igmp

import (
	"net"
	"testing"
)

func TestLeaveGroup_MarshalRoundTrip(t *testing.T) {
	original := &LeaveGroup{
		GroupAddress: net.IPv4(239, 1, 1, 1),
	}

	data, err := original.Marshal()
	if err != nil {
		t.Fatalf("Marshal() error: %v", err)
	}

	if len(data) != 8 {
		t.Fatalf("Marshal() length = %d, want 8", len(data))
	}
	if data[0] != TypeLeaveGroupV2 {
		t.Errorf("type = 0x%02x, want 0x%02x", data[0], TypeLeaveGroupV2)
	}

	parsed, err := ParseLeaveGroup(data)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	if parsed.Version() != 2 {
		t.Errorf("Version() = %d, want 2", parsed.Version())
	}
	if parsed.Type() != TypeLeaveGroupV2 {
		t.Errorf("Type() = 0x%02x, want 0x%02x", parsed.Type(), TypeLeaveGroupV2)
	}
	if !parsed.GroupAddress.Equal(original.GroupAddress.To4()) {
		t.Errorf("GroupAddress = %v, want %v", parsed.GroupAddress, original.GroupAddress)
	}
}

func TestLeaveGroup_InvalidIP(t *testing.T) {
	l := &LeaveGroup{GroupAddress: nil}
	_, err := l.Marshal()
	if err == nil {
		t.Error("expected error for nil group address")
	}

	l = &LeaveGroup{GroupAddress: net.ParseIP("::1")}
	_, err = l.Marshal()
	if err == nil {
		t.Error("expected error for IPv6 group address")
	}
}

func TestLeaveGroup_TooShort(t *testing.T) {
	_, err := ParseLeaveGroup([]byte{0x17, 0x00, 0x00})
	if err == nil {
		t.Error("expected error for short data")
	}
}

func TestLeaveGroup_WrongType(t *testing.T) {
	data := []byte{0x11, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	_, err := ParseLeaveGroup(data)
	if err == nil {
		t.Error("expected error for wrong type")
	}
}

func TestLeaveGroup_BadChecksum(t *testing.T) {
	data := []byte{0x17, 0x00, 0xff, 0xff, 0xef, 0x01, 0x01, 0x01}
	_, err := ParseLeaveGroup(data)
	if err == nil {
		t.Error("expected error for bad checksum")
	}
}

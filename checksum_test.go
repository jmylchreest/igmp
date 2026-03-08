// SPDX-License-Identifier: MIT

package igmp

import "testing"

func TestChecksum(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		initial  uint32
		expected uint16
	}{
		{
			name:     "all zeros",
			data:     []byte{0x00, 0x00, 0x00, 0x00},
			expected: 0xffff,
		},
		{
			name:     "all ones",
			data:     []byte{0xff, 0xff, 0xff, 0xff},
			expected: 0x0000,
		},
		{
			name: "RFC 1071 example",
			// 0x0001 + 0xf203 + 0xf4f5 + 0xf6f7 = 0x2ddf0, folded = 0xddf2, ~= 0x220d
			data:     []byte{0x00, 0x01, 0xf2, 0x03, 0xf4, 0xf5, 0xf6, 0xf7},
			expected: 0x220d,
		},
		{
			name: "odd length",
			// 0x0102 + 0x0300 (padded) = 0x0402, ~= 0xfbfd
			data:     []byte{0x01, 0x02, 0x03},
			expected: 0xfbfd,
		},
		{
			name:     "single byte",
			data:     []byte{0xff},
			expected: 0x00ff,
		},
		{
			name:     "empty data",
			data:     []byte{},
			expected: 0xffff,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Checksum(tt.data, tt.initial)
			if got != tt.expected {
				t.Errorf("Checksum(%x, %d) = 0x%04x, want 0x%04x",
					tt.data, tt.initial, got, tt.expected)
			}
		})
	}
}

func TestValidateChecksum(t *testing.T) {
	// Build a simple IGMPv2 query and verify its checksum is valid.
	// Type=0x11, MaxResp=100 (0x64), Checksum=?, Group=0.0.0.0
	data := []byte{0x11, 0x64, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	// Compute and embed the checksum.
	csum := Checksum(data, 0)
	data[2] = byte(csum >> 8)
	data[3] = byte(csum)

	if !ValidateChecksum(data) {
		t.Error("ValidateChecksum returned false for valid checksum")
	}

	// Corrupt a byte.
	data[4] = 0xff
	if ValidateChecksum(data) {
		t.Error("ValidateChecksum returned true for corrupted data")
	}
}

func TestChecksumRoundTrip(t *testing.T) {
	// Verifying that computing the checksum and including it in the data
	// produces a zero result when re-checksummed.
	data := []byte{0x11, 0x64, 0x00, 0x00, 0xe0, 0x00, 0x00, 0x01}
	csum := Checksum(data, 0)
	data[2] = byte(csum >> 8)
	data[3] = byte(csum)

	if Checksum(data, 0) != 0 {
		t.Errorf("expected checksum of checksummed data to be 0, got 0x%04x", Checksum(data, 0))
	}
}

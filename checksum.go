// SPDX-License-Identifier: MIT

package igmp

// Checksum computes the Internet checksum (RFC 1071) over the given data.
// This is used for IGMP message integrity verification.
//
// The initial parameter allows chaining checksums over non-contiguous data;
// pass 0 for a standalone checksum.
func Checksum(data []byte, initial uint32) uint16 {
	csum := initial

	// Sum 16-bit words.
	length := len(data)
	for i := 0; i+1 < length; i += 2 {
		csum += uint32(data[i])<<8 | uint32(data[i+1])
	}

	// If odd length, pad the last byte with zero.
	if length%2 == 1 {
		csum += uint32(data[length-1]) << 8
	}

	// Fold 32-bit sum into 16 bits.
	for csum > 0xffff {
		csum = (csum >> 16) + (csum & 0xffff)
	}

	return ^uint16(csum)
}

// ValidateChecksum verifies that the Internet checksum of data is valid.
// A valid message checksums to zero when the checksum field is included.
func ValidateChecksum(data []byte) bool {
	return Checksum(data, 0) == 0
}

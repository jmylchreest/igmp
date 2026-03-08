// SPDX-License-Identifier: MIT

package igmp

import "fmt"

// Parse decodes raw IGMP payload bytes into the appropriate Message type.
//
// It inspects the first byte (type field) and the message length to
// determine the IGMP version and dispatch to the correct parser.
//
// The data should contain only the IGMP payload, not the IP header.
func Parse(data []byte) (Message, error) {
	if len(data) < 8 {
		return nil, fmt.Errorf("%w: need at least 8 bytes, got %d", ErrTooShort, len(data))
	}

	msgType := data[0]

	switch msgType {
	case TypeMembershipQuery:
		return ParseMembershipQuery(data)
	case TypeMembershipReportV1:
		return ParseMembershipReportV1(data)
	case TypeMembershipReportV2:
		return ParseMembershipReportV2(data)
	case TypeLeaveGroupV2:
		return ParseLeaveGroup(data)
	case TypeMembershipReportV3:
		return ParseMembershipReportV3(data)
	default:
		return nil, fmt.Errorf("%w: 0x%02x", ErrInvalidType, msgType)
	}
}

// SPDX-License-Identifier: MIT

// Package igmp provides a comprehensive, spec-compliant implementation of the
// Internet Group Management Protocol (IGMP) for Go.
//
// It supports IGMPv1 (RFC 1112), IGMPv2 (RFC 2236), and IGMPv3 (RFC 3376)
// message types with full marshal/unmarshal capabilities, a raw socket
// connection abstraction, a periodic querier, and a passive listener.
//
// # Library Usage
//
// The root package is the library. Import it as:
//
//	import "github.com/jmylchreest/igmp"
//
// Parse incoming IGMP messages:
//
//	msg, err := igmp.Parse(data)
//
// Construct and send queries:
//
//	q := &igmp.MembershipQuery{
//	    GroupAddress: net.IPv4zero,
//	    MaxRespTime: 10 * time.Second,
//	}
//	data, err := q.Marshal()
//
// # CLI Tools
//
// Two CLI tools are provided:
//   - cmd/igmpqd: A lightweight IGMPv2/v3 query daemon
//   - cmd/igmpmon: A real-time IGMP traffic monitor with TUI dashboard
package igmp

import "net"

// IGMP message type codes as defined in the IANA registry.
// See: https://www.iana.org/assignments/igmp-type-numbers/igmp-type-numbers.xhtml
const (
	TypeMembershipQuery    uint8 = 0x11 // RFC 1112/2236/3376
	TypeMembershipReportV1 uint8 = 0x12 // RFC 1112
	TypeMembershipReportV2 uint8 = 0x16 // RFC 2236
	TypeLeaveGroupV2       uint8 = 0x17 // RFC 2236
	TypeMembershipReportV3 uint8 = 0x22 // RFC 3376/9776
)

// IGMPv3 Group Record types as defined in RFC 3376 Section 4.2.12.
const (
	RecordModeIsInclude   uint8 = 1 // Current-State Record
	RecordModeIsExclude   uint8 = 2 // Current-State Record
	RecordChangeToInclude uint8 = 3 // Filter-Mode-Change Record
	RecordChangeToExclude uint8 = 4 // Filter-Mode-Change Record
	RecordAllowNewSources uint8 = 5 // Source-List-Change Record
	RecordBlockOldSources uint8 = 6 // Source-List-Change Record
)

// Protocol constants per RFCs.
const (
	// ProtocolNumber is the IP protocol number for IGMP.
	ProtocolNumber = 2

	// DefaultTTL is the default IP TTL for IGMP messages (RFC 2236 Section 2).
	DefaultTTL = 1

	// DefaultTOS is the default IP TOS for IGMP queries (DSCP CS6).
	DefaultTOS = 0xc0

	// DefaultMaxResponseTime is the default Max Response Time in 1/10 second
	// units for IGMPv2 queries (RFC 2236 Section 8.3). This equals 10 seconds.
	DefaultMaxResponseTime = 100

	// DefaultQueryInterval is the default interval between General Queries
	// in seconds (RFC 3376 Section 8.2).
	DefaultQueryInterval = 125

	// DefaultRobustnessVariable is the default Robustness Variable
	// (RFC 3376 Section 8.1).
	DefaultRobustnessVariable = 2

	// HeaderLen is the minimum IGMP message length in bytes (v1/v2).
	HeaderLen = 8
)

// Well-known multicast addresses used by IGMP.
var (
	// AllHosts is the all-hosts multicast address (224.0.0.1).
	// General Queries are sent to this address.
	AllHosts = net.IPv4(224, 0, 0, 1)

	// AllRouters is the all-routers multicast address (224.0.0.2).
	// IGMPv2 Leave Group messages are sent to this address.
	AllRouters = net.IPv4(224, 0, 0, 2)

	// IGMPv3ReportAddress is the IGMPv3 report destination address (224.0.0.22).
	// IGMPv3 Membership Reports are sent to this address.
	IGMPv3ReportAddress = net.IPv4(224, 0, 0, 22)
)

// RecordTypeName returns the human-readable name for an IGMPv3 Group Record type.
func RecordTypeName(rt uint8) string {
	switch rt {
	case RecordModeIsInclude:
		return "MODE_IS_INCLUDE"
	case RecordModeIsExclude:
		return "MODE_IS_EXCLUDE"
	case RecordChangeToInclude:
		return "CHANGE_TO_INCLUDE_MODE"
	case RecordChangeToExclude:
		return "CHANGE_TO_EXCLUDE_MODE"
	case RecordAllowNewSources:
		return "ALLOW_NEW_SOURCES"
	case RecordBlockOldSources:
		return "BLOCK_OLD_SOURCES"
	default:
		return "UNKNOWN"
	}
}

// MessageTypeName returns the human-readable name for an IGMP message type.
func MessageTypeName(t uint8) string {
	switch t {
	case TypeMembershipQuery:
		return "Membership Query"
	case TypeMembershipReportV1:
		return "IGMPv1 Membership Report"
	case TypeMembershipReportV2:
		return "IGMPv2 Membership Report"
	case TypeLeaveGroupV2:
		return "IGMPv2 Leave Group"
	case TypeMembershipReportV3:
		return "IGMPv3 Membership Report"
	default:
		return "Unknown"
	}
}

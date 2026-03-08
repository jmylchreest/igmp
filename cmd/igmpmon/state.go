// SPDX-License-Identifier: MIT

package main

import (
	"net"
	"sync"
	"time"

	"github.com/jmylchreest/igmp"
)

// maxPacketLog is the maximum number of packets kept in the log.
const maxPacketLog = 500

// groupStaleTimeout is how long after the last report a group is marked stale.
const groupStaleTimeout = 5 * time.Minute

// memberStaleTimeout is how long after the last report a member is marked stale.
const memberStaleTimeout = 2 * time.Minute

// packetEntry represents a single IGMP packet in the log.
type packetEntry struct {
	Time     time.Time
	TypeName string
	Version  int
	Source   net.IP
	Group    net.IP
	Detail   string
}

// memberState tracks a single member of a multicast group.
type memberState struct {
	Address    net.IP
	Version    int
	FilterMode string // "INCLUDE", "EXCLUDE", or "" (v2)
	Sources    []net.IP
	LastSeen   time.Time
	LeftAt     *time.Time
}

// groupState tracks an active multicast group.
type groupState struct {
	GroupAddress net.IP
	Members      map[string]*memberState // keyed by IP string
	LastReport   time.Time
	LastVersion  int
}

// querierState tracks the active querier.
type querierState struct {
	Address     net.IP
	Version     int
	LastSeen    time.Time
	Interval    time.Duration
	MaxRespTime time.Duration
}

// stats tracks packet counters.
type stats struct {
	TotalPackets int
	Queries      int
	Reports      int
	Leaves       int
	V1Count      int
	V2Count      int
	V3Count      int
	StartTime    time.Time
}

// monitorState is the central state model for the monitor.
type monitorState struct {
	mu      sync.Mutex
	stats   stats
	groups  map[string]*groupState // keyed by group IP string
	querier *querierState
	packets []packetEntry
	iface   string
}

func newMonitorState(iface string) *monitorState {
	return &monitorState{
		stats: stats{
			StartTime: time.Now(),
		},
		groups: make(map[string]*groupState),
		iface:  iface,
	}
}

// processMessage updates the state with a received IGMP message.
func (s *monitorState) processMessage(rm *igmp.ReceivedMessage) {
	s.mu.Lock()
	defer s.mu.Unlock()

	msg := rm.Message
	s.stats.TotalPackets++

	switch msg.Version() {
	case 1:
		s.stats.V1Count++
	case 2:
		s.stats.V2Count++
	case 3:
		s.stats.V3Count++
	}

	entry := packetEntry{
		Time:     rm.Received,
		TypeName: igmp.MessageTypeName(msg.Type()),
		Version:  msg.Version(),
		Source:   rm.Source,
	}

	switch m := msg.(type) {
	case *igmp.MembershipQuery:
		s.stats.Queries++
		entry.Group = m.GroupAddress
		entry.Detail = queryDetail(m)
		s.updateQuerier(rm.Source, m)

	case *igmp.MembershipReportV1:
		s.stats.Reports++
		entry.Group = m.GroupAddress
		s.updateGroupMember(m.GroupAddress, rm.Source, 1, "", nil)

	case *igmp.MembershipReportV2:
		s.stats.Reports++
		entry.Group = m.GroupAddress
		s.updateGroupMember(m.GroupAddress, rm.Source, 2, "", nil)

	case *igmp.LeaveGroup:
		s.stats.Leaves++
		entry.Group = m.GroupAddress
		entry.Detail = "LEAVE"
		s.markMemberLeft(m.GroupAddress, rm.Source)

	case *igmp.MembershipReportV3:
		s.stats.Reports++
		for _, gr := range m.GroupRecords {
			entry.Group = gr.GroupAddress
			filterMode := ""
			switch gr.RecordType {
			case igmp.RecordModeIsInclude, igmp.RecordChangeToInclude:
				filterMode = "INCLUDE"
			case igmp.RecordModeIsExclude, igmp.RecordChangeToExclude:
				filterMode = "EXCLUDE"
			}

			// A CHANGE_TO_INCLUDE with empty source list is effectively a leave.
			if gr.RecordType == igmp.RecordChangeToInclude && len(gr.SourceAddresses) == 0 {
				s.markMemberLeft(gr.GroupAddress, rm.Source)
				entry.Detail = "LEAVE (v3)"
			} else {
				s.updateGroupMember(gr.GroupAddress, rm.Source, 3, filterMode, gr.SourceAddresses)
				entry.Detail = igmp.RecordTypeName(gr.RecordType)
			}
		}
	}

	// Append to packet log, trim if too large.
	s.packets = append(s.packets, entry)
	if len(s.packets) > maxPacketLog {
		s.packets = s.packets[len(s.packets)-maxPacketLog:]
	}
}

func (s *monitorState) updateQuerier(src net.IP, q *igmp.MembershipQuery) {
	s.querier = &querierState{
		Address:     src,
		Version:     q.Version(),
		LastSeen:    time.Now(),
		MaxRespTime: q.MaxRespTime,
	}
	if q.QQIC > 0 {
		s.querier.Interval = q.QQIC
	}
}

func (s *monitorState) updateGroupMember(group, src net.IP, ver int, filterMode string, sources []net.IP) {
	key := group.String()
	g, ok := s.groups[key]
	if !ok {
		g = &groupState{
			GroupAddress: copyIP(group),
			Members:      make(map[string]*memberState),
		}
		s.groups[key] = g
	}
	g.LastReport = time.Now()
	g.LastVersion = ver

	memberKey := src.String()
	member, ok := g.Members[memberKey]
	if !ok {
		member = &memberState{
			Address: copyIP(src),
		}
		g.Members[memberKey] = member
	}
	member.Version = ver
	member.FilterMode = filterMode
	member.Sources = copyIPs(sources)
	member.LastSeen = time.Now()
	member.LeftAt = nil // They're back if they were marked as left.
}

func (s *monitorState) markMemberLeft(group, src net.IP) {
	key := group.String()
	g, ok := s.groups[key]
	if !ok {
		return
	}

	memberKey := src.String()
	member, ok := g.Members[memberKey]
	if !ok {
		return
	}

	now := time.Now()
	member.LeftAt = &now
}

// snapshot returns a copy of the state for rendering (avoids holding the lock).
func (s *monitorState) snapshot() (stats, map[string]*groupState, *querierState, []packetEntry) {
	s.mu.Lock()
	defer s.mu.Unlock()

	st := s.stats

	groups := make(map[string]*groupState, len(s.groups))
	for k, v := range s.groups {
		groups[k] = v
	}

	var q *querierState
	if s.querier != nil {
		qCopy := *s.querier
		q = &qCopy
	}

	packets := make([]packetEntry, len(s.packets))
	copy(packets, s.packets)

	return st, groups, q, packets
}

func queryDetail(q *igmp.MembershipQuery) string {
	if q.IsGeneral() {
		return "General"
	}
	if q.IsGroupAndSourceSpecific() {
		return "Group-and-Source-Specific"
	}
	return "Group-Specific"
}

func copyIP(ip net.IP) net.IP {
	if ip == nil {
		return nil
	}
	c := make(net.IP, len(ip))
	copy(c, ip)
	return c
}

func copyIPs(ips []net.IP) []net.IP {
	if ips == nil {
		return nil
	}
	c := make([]net.IP, len(ips))
	for i, ip := range ips {
		c[i] = copyIP(ip)
	}
	return c
}

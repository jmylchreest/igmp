// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jmylchreest/igmp"
	"github.com/spf13/cobra"
)

// Styles
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("15")).
			Background(lipgloss.Color("62")).
			Padding(0, 1)

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("39"))

	statLabelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245"))

	statValueStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("15"))

	groupActiveStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("82"))

	groupStaleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214"))

	queryStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("39"))

	reportStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("82"))

	leaveStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214"))

	selectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("15")).
			Background(lipgloss.Color("62"))

	borderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245"))

	warningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214")).
			Italic(true)
)

type pane int

const (
	paneGroups pane = iota
	panePackets
	paneMembers
)

// keyMap defines keybindings.
type keyMap struct {
	Up     key.Binding
	Down   key.Binding
	Enter  key.Binding
	Escape key.Binding
	Tab    key.Binding
	Freeze key.Binding
	Quit   key.Binding
}

var keys = keyMap{
	Up:     key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("up/k", "up")),
	Down:   key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("dn/j", "down")),
	Enter:  key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "drill into group")),
	Escape: key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
	Tab:    key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "switch pane")),
	Freeze: key.NewBinding(key.WithKeys("f"), key.WithHelp("f", "freeze/unfreeze")),
	Quit:   key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Tab, k.Enter, k.Escape, k.Freeze, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{k.ShortHelp()}
}

// tickMsg triggers a UI refresh.
type tickMsg time.Time

// igmpMsg wraps a received IGMP message.
type igmpMsg *igmp.ReceivedMessage

// model is the Bubble Tea model.
type model struct {
	state  *monitorState
	cancel context.CancelFunc
	width  int
	height int
	active pane
	frozen bool
	help   help.Model

	// Groups pane state
	groupKeys     []string // sorted group keys
	groupCursor   int
	selectedGroup string // group key when drilled in

	// Packet log viewport
	packetViewport viewport.Model

	// Error state
	err error
}

func initialModel(state *monitorState, cancel context.CancelFunc) model {
	h := help.New()
	h.ShortSeparator = "  "

	return model{
		state:          state,
		cancel:         cancel,
		active:         paneGroups,
		help:           h,
		packetViewport: viewport.New(80, 10),
	}
}

func (m model) Init() tea.Cmd {
	return tickCmd()
}

func tickCmd() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Quit):
			m.cancel()
			return m, tea.Quit

		case key.Matches(msg, keys.Tab):
			if m.selectedGroup != "" {
				// In members view, tab does nothing special
			} else {
				if m.active == paneGroups {
					m.active = panePackets
				} else {
					m.active = paneGroups
				}
			}

		case key.Matches(msg, keys.Freeze):
			m.frozen = !m.frozen

		case key.Matches(msg, keys.Up):
			if m.active == paneGroups && m.selectedGroup == "" {
				if m.groupCursor > 0 {
					m.groupCursor--
				}
			} else if m.active == panePackets || m.selectedGroup != "" {
				m.packetViewport.LineUp(1)
			}

		case key.Matches(msg, keys.Down):
			if m.active == paneGroups && m.selectedGroup == "" {
				if m.groupCursor < len(m.groupKeys)-1 {
					m.groupCursor++
				}
			} else if m.active == panePackets || m.selectedGroup != "" {
				m.packetViewport.LineDown(1)
			}

		case key.Matches(msg, keys.Enter):
			if m.active == paneGroups && m.selectedGroup == "" && len(m.groupKeys) > 0 {
				m.selectedGroup = m.groupKeys[m.groupCursor]
				m.active = paneMembers
			}

		case key.Matches(msg, keys.Escape):
			if m.selectedGroup != "" {
				m.selectedGroup = ""
				m.active = paneGroups
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.packetViewport.Width = msg.Width - 4
		m.packetViewport.Height = max(msg.Height/3-2, 5)

	case tickMsg:
		if !m.frozen {
			m.refreshGroupKeys()
		}
		return m, tickCmd()
	}

	return m, nil
}

func (m *model) refreshGroupKeys() {
	_, groups, _, _ := m.state.snapshot()
	keys := make([]string, 0, len(groups))
	for k := range groups {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	m.groupKeys = keys
	if m.groupCursor >= len(m.groupKeys) {
		m.groupCursor = max(len(m.groupKeys)-1, 0)
	}
}

func (m model) View() string {
	if m.width == 0 {
		return "Initializing..."
	}

	if m.selectedGroup != "" {
		return m.viewMembers()
	}

	st, groups, querier, packets := m.state.snapshot()

	var sections []string

	// Title bar
	title := titleStyle.Render(fmt.Sprintf(" igmpmon  -  IGMP Monitor  |  %s ", m.state.iface))
	frozen := ""
	if m.frozen {
		frozen = warningStyle.Render("  [FROZEN]")
	}
	sections = append(sections, title+frozen)

	// Stats + Querier row
	sections = append(sections, m.renderStats(st, querier))

	// Groups pane
	sections = append(sections, m.renderGroups(groups))

	// Packet log
	sections = append(sections, m.renderPacketLog(packets))

	// Help
	sections = append(sections, helpStyle.Render(m.help.ShortHelpView(keys.ShortHelp())))

	return strings.Join(sections, "\n")
}

func (m model) renderStats(st stats, querier *querierState) string {
	elapsed := time.Since(st.StartTime).Truncate(time.Second)

	left := fmt.Sprintf(
		"%s %s  %s %s  %s %s  %s %s",
		statLabelStyle.Render("Packets:"), statValueStyle.Render(fmt.Sprintf("%d", st.TotalPackets)),
		statLabelStyle.Render("Queries:"), statValueStyle.Render(fmt.Sprintf("%d", st.Queries)),
		statLabelStyle.Render("Reports:"), statValueStyle.Render(fmt.Sprintf("%d", st.Reports)),
		statLabelStyle.Render("Leaves:"), statValueStyle.Render(fmt.Sprintf("%d", st.Leaves)),
	)

	versions := fmt.Sprintf(
		"%s %s  %s %s  %s %s  %s %s",
		statLabelStyle.Render("v1:"), statValueStyle.Render(fmt.Sprintf("%d", st.V1Count)),
		statLabelStyle.Render("v2:"), statValueStyle.Render(fmt.Sprintf("%d", st.V2Count)),
		statLabelStyle.Render("v3:"), statValueStyle.Render(fmt.Sprintf("%d", st.V3Count)),
		statLabelStyle.Render("Duration:"), statValueStyle.Render(elapsed.String()),
	)

	qInfo := statLabelStyle.Render("Querier: ") + statValueStyle.Render("none detected")
	if querier != nil {
		ago := time.Since(querier.LastSeen).Truncate(time.Second)
		qInfo = fmt.Sprintf("%s %s %s %s",
			statLabelStyle.Render("Querier:"),
			statValueStyle.Render(fmt.Sprintf("%s (v%d)", querier.Address, querier.Version)),
			statLabelStyle.Render("Last:"),
			statValueStyle.Render(fmt.Sprintf("%s ago", ago)),
		)
	}

	content := left + "\n" + versions + "\n" + qInfo
	return borderStyle.Width(m.width - 2).Render(
		headerStyle.Render("STATISTICS") + "\n" + content,
	)
}

func (m model) renderGroups(groups map[string]*groupState) string {
	paneMark := ""
	if m.active == paneGroups {
		paneMark = " *"
	}
	header := headerStyle.Render("ACTIVE GROUPS"+paneMark) + "                                    " +
		statLabelStyle.Render("[Enter] drill into group")

	if len(m.groupKeys) == 0 {
		content := statLabelStyle.Render("  No active groups detected yet...")
		return borderStyle.Width(m.width - 2).Render(header + "\n" + content)
	}

	colHeader := fmt.Sprintf("  %-18s %-8s %-14s %-6s %-8s",
		"Group", "Members", "Last Report", "Ver", "Status")
	lines := []string{statLabelStyle.Render(colHeader)}

	now := time.Now()
	hasV2 := false
	for _, key := range m.groupKeys {
		g, ok := groups[key]
		if !ok {
			continue
		}

		activeMemberCount := 0
		for _, member := range g.Members {
			if member.LeftAt == nil {
				activeMemberCount++
			}
		}

		ago := now.Sub(g.LastReport).Truncate(time.Second)
		status := "active"
		style := groupActiveStyle
		if ago > groupStaleTimeout {
			status = "stale"
			style = groupStaleStyle
		}

		suffix := ""
		if g.LastVersion <= 2 {
			suffix = "*"
			hasV2 = true
		}

		cursor := "  "
		lineStyle := style
		if m.active == paneGroups && m.groupCursor < len(m.groupKeys) && m.groupKeys[m.groupCursor] == key {
			cursor = "> "
			lineStyle = selectedStyle
		}

		line := fmt.Sprintf("%s%-18s %-8s %-14s v%-5d %-8s",
			cursor,
			g.GroupAddress,
			fmt.Sprintf("%d%s", activeMemberCount, suffix),
			fmt.Sprintf("%s ago", ago),
			g.LastVersion,
			status,
		)
		lines = append(lines, lineStyle.Render(line))
	}

	if hasV2 {
		lines = append(lines, warningStyle.Render("  * = IGMPv2: count may be understated (report suppression)"))
	}

	content := strings.Join(lines, "\n")
	return borderStyle.Width(m.width - 2).Render(header + "\n" + content)
}

func (m model) renderPacketLog(packets []packetEntry) string {
	paneMark := ""
	if m.active == panePackets {
		paneMark = " *"
	}
	frozenMark := ""
	if m.frozen {
		frozenMark = "  " + warningStyle.Render("[frozen]")
	}
	header := headerStyle.Render("PACKET LOG"+paneMark) + frozenMark

	if len(packets) == 0 {
		content := statLabelStyle.Render("  Waiting for IGMP packets...")
		return borderStyle.Width(m.width - 2).Render(header + "\n" + content)
	}

	colHeader := fmt.Sprintf("  %-10s %-24s %-16s %-18s %-s",
		"Time", "Type", "Source", "Group", "Detail")
	lines := []string{statLabelStyle.Render(colHeader)}

	// Show the last N packets that fit.
	maxLines := max(m.height/3-2, 5)
	start := len(packets) - maxLines
	if start < 0 {
		start = 0
	}

	for _, p := range packets[start:] {
		timeStr := p.Time.Format("15:04:05")
		style := statLabelStyle
		switch {
		case strings.Contains(p.TypeName, "Query"):
			style = queryStyle
		case strings.Contains(p.TypeName, "Report"):
			style = reportStyle
		case strings.Contains(p.TypeName, "Leave"):
			style = leaveStyle
		}

		groupStr := ""
		if p.Group != nil {
			groupStr = p.Group.String()
		}
		sourceStr := ""
		if p.Source != nil {
			sourceStr = p.Source.String()
		}

		line := fmt.Sprintf("  %-10s %-24s %-16s %-18s %-s",
			timeStr, p.TypeName, sourceStr, groupStr, p.Detail)
		lines = append(lines, style.Render(line))
	}

	content := strings.Join(lines, "\n")
	return borderStyle.Width(m.width - 2).Render(header + "\n" + content)
}

func (m model) viewMembers() string {
	_, groups, _, _ := m.state.snapshot()

	g, ok := groups[m.selectedGroup]
	if !ok {
		return "Group not found. Press Esc to go back."
	}

	var sections []string

	// Title
	title := titleStyle.Render(fmt.Sprintf(" GROUP: %s ", m.selectedGroup))
	back := helpStyle.Render("  [Esc] back")
	sections = append(sections, title+back)

	// Members table
	memberHeader := headerStyle.Render("MEMBERS")
	colHeader := fmt.Sprintf("  %-18s %-6s %-10s %-24s %-14s",
		"IP", "Ver", "Filter", "Sources", "Last Seen")
	memberLines := []string{statLabelStyle.Render(colHeader)}

	// Sort members by IP.
	memberKeys := make([]string, 0, len(g.Members))
	for k := range g.Members {
		memberKeys = append(memberKeys, k)
	}
	sort.Strings(memberKeys)

	now := time.Now()
	for _, mk := range memberKeys {
		member := g.Members[mk]
		ago := now.Sub(member.LastSeen).Truncate(time.Second)

		status := fmt.Sprintf("%s ago", ago)
		style := groupActiveStyle
		if member.LeftAt != nil {
			status = fmt.Sprintf("left (%s)", member.LeftAt.Format("15:04:05"))
			style = leaveStyle
		} else if ago > memberStaleTimeout {
			status += "  !!"
			style = groupStaleStyle
		}

		filterMode := member.FilterMode
		if filterMode == "" {
			filterMode = "-"
		}

		sourcesStr := "*"
		if len(member.Sources) > 0 {
			parts := make([]string, len(member.Sources))
			for i, s := range member.Sources {
				parts[i] = s.String()
			}
			sourcesStr = strings.Join(parts, ",")
		}

		line := fmt.Sprintf("  %-18s v%-5d %-10s %-24s %-14s",
			member.Address, member.Version, filterMode, sourcesStr, status)
		memberLines = append(memberLines, style.Render(line))
	}

	hasV2 := g.LastVersion <= 2
	if hasV2 {
		memberLines = append(memberLines, "")
		memberLines = append(memberLines, warningStyle.Render("  IGMPv2: report suppression may hide additional members"))
	}

	memberContent := strings.Join(memberLines, "\n")
	sections = append(sections, borderStyle.Width(m.width-2).Render(memberHeader+"\n"+memberContent))

	// Group packet history (filtered)
	_, _, _, packets := m.state.snapshot()
	histHeader := headerStyle.Render("GROUP HISTORY")
	histColHeader := fmt.Sprintf("  %-10s %-24s %-16s %-s",
		"Time", "Type", "Source", "Detail")
	histLines := []string{statLabelStyle.Render(histColHeader)}

	for _, p := range packets {
		if p.Group == nil || p.Group.String() != m.selectedGroup {
			continue
		}
		timeStr := p.Time.Format("15:04:05")
		style := statLabelStyle
		switch {
		case strings.Contains(p.TypeName, "Query"):
			style = queryStyle
		case strings.Contains(p.TypeName, "Report"):
			style = reportStyle
		case strings.Contains(p.TypeName, "Leave"):
			style = leaveStyle
		}

		sourceStr := ""
		if p.Source != nil {
			sourceStr = p.Source.String()
		}

		line := fmt.Sprintf("  %-10s %-24s %-16s %-s",
			timeStr, p.TypeName, sourceStr, p.Detail)
		histLines = append(histLines, style.Render(line))
	}

	if len(histLines) == 1 {
		histLines = append(histLines, statLabelStyle.Render("  No packets for this group yet..."))
	}

	histContent := strings.Join(histLines, "\n")
	sections = append(sections, borderStyle.Width(m.width-2).Render(histHeader+"\n"+histContent))

	// Help
	sections = append(sections, helpStyle.Render(m.help.ShortHelpView(keys.ShortHelp())))

	return strings.Join(sections, "\n")
}

// runMonitor is the Cobra RunE handler that starts the TUI.
func runMonitor(cmd *cobra.Command, _ []string) error {
	iface, _ := cmd.Flags().GetString("interface")
	bufSize, _ := cmd.Flags().GetInt("buffer")

	state := newMonitorState(iface)
	logger := slog.New(slog.DiscardHandler)

	listener, err := igmp.NewListener(igmp.ListenerConfig{
		Interface:  iface,
		Logger:     logger,
		BufferSize: bufSize,
	})
	if err != nil {
		return fmt.Errorf("failed to create listener: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	msgCh, err := listener.Listen(ctx)
	if err != nil {
		cancel()
		return fmt.Errorf("failed to start listener: %w", err)
	}

	// Process incoming IGMP messages in a goroutine.
	go func() {
		for msg := range msgCh {
			state.processMessage(msg)
		}
	}()

	p := tea.NewProgram(
		initialModel(state, cancel),
		tea.WithAltScreen(),
	)

	_, err = p.Run()
	cancel()
	listener.Close()
	return err
}

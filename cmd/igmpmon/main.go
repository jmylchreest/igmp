// SPDX-License-Identifier: MIT

// igmpmon is a real-time IGMP traffic monitor with a TUI dashboard.
//
// It passively listens for IGMP messages on a network interface and displays
// statistics, active multicast groups, group membership, and a live packet log.
//
// Install:
//
//	go install github.com/jmylchreest/igmp/cmd/igmpmon@latest
//
// Usage:
//
//	igmpmon --interface eth0
//	igmpmon -I eth0
package main

import (
	"fmt"
	"os"

	"github.com/jmylchreest/igmp/internal/version"
	"github.com/spf13/cobra"
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "igmpmon",
	Short: "Real-time IGMP traffic monitor with TUI dashboard",
	Long: `igmpmon passively listens for IGMP messages on a network interface
and displays a real-time dashboard with:

  - Statistics: packet counts, version breakdown, session duration
  - Active Groups: multicast groups with member counts and status
  - Group Members: drill-down view showing individual reporters per group
  - Packet Log: scrollable live feed of all IGMP packets

Requires elevated privileges (root or CAP_NET_RAW) to open a raw socket.`,
	RunE: runMonitor,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, _ []string) {
		fmt.Println(version.Info())
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)

	flags := rootCmd.Flags()
	flags.StringP("interface", "I", "", "Network interface to monitor (required)")
	flags.Int("buffer", 256, "Listener channel buffer size")
}

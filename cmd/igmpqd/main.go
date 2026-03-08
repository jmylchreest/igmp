// SPDX-License-Identifier: MIT

// igmpqd is a lightweight IGMP query daemon that sends periodic IGMPv2 or
// IGMPv3 Membership Query messages on a network interface.
//
// Install:
//
//	go install github.com/jmylchreest/igmp/cmd/igmpqd@latest
//
// Usage:
//
//	igmpqd run --interface eth0
//	igmpqd run --interface eth0 --version 3 --interval 60
//	igmpqd version
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jmylchreest/igmp"
	"github.com/jmylchreest/igmp/internal/version"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "igmpqd",
	Short: "igmpqd is a lightweight IGMP query daemon.",
	Long: `igmpqd is a lightweight and simple IGMP query daemon, designed to
send periodic IGMPv2/v3 membership queries. It is useful in environments
that don't have a dedicated IGMP querier mechanism.

The IGMP library is available at: github.com/jmylchreest/igmp`,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Println(version.Info())
	},
}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Start the IGMP query daemon",
	Long: `Start sending periodic IGMP membership queries on the specified
network interface. The daemon runs until interrupted (SIGINT/SIGTERM).`,
	RunE: runQuerier,
}

func init() {
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(runCmd)

	flags := runCmd.Flags()

	// Primary names (kebab-case).
	flags.StringP("interface", "I", "", "Network interface to send queries on")
	flags.StringP("grp-address", "g", "0.0.0.0", "Group address for the IGMP query")
	flags.StringP("dst-address", "d", "224.0.0.1", "Destination IP for the IGMP query")
	flags.IntP("interval", "i", igmp.DefaultQueryInterval, "Seconds between query messages")
	flags.IntP("ttl", "t", igmp.DefaultTTL, "IP TTL of the IGMP query")
	flags.IntP("max-response-time", "m", igmp.DefaultMaxResponseTime, "Max response time in 1/10 sec units")
	flags.IntP("version", "V", 2, "IGMP version to use (2 or 3)")
	flags.Bool("debug", false, "Enable debug logging to stderr")
	flags.Bool("no-router-alert", false, "Disable the IP Router Alert option")

	// Backward-compatible hidden aliases (camelCase).
	flags.String("grpAddress", "0.0.0.0", "Alias for --grp-address")
	flags.String("dstAddress", "224.0.0.1", "Alias for --dst-address")
	flags.Int("maxResponseTime", igmp.DefaultMaxResponseTime, "Alias for --max-response-time")
	_ = flags.MarkHidden("grpAddress")
	_ = flags.MarkHidden("dstAddress")
	_ = flags.MarkHidden("maxResponseTime")
}

func runQuerier(cmd *cobra.Command, _ []string) error {
	flags := cmd.Flags()

	debug, _ := flags.GetBool("debug")
	logLevel := slog.LevelInfo
	if debug {
		logLevel = slog.LevelDebug
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel}))

	// Resolve flags with backward-compatible aliases.
	grpAddress := flagStringWithAlias(flags, "grp-address", "grpAddress")
	dstAddress := flagStringWithAlias(flags, "dst-address", "dstAddress")
	maxRespTime := flagIntWithAlias(flags, "max-response-time", "maxResponseTime")

	ifaceName, _ := flags.GetString("interface")
	interval, _ := flags.GetInt("interval")
	ttl, _ := flags.GetInt("ttl")
	igmpVersion, _ := flags.GetInt("version")
	noRouterAlert, _ := flags.GetBool("no-router-alert")

	grpIP := net.ParseIP(grpAddress)
	if grpIP == nil {
		return fmt.Errorf("invalid group address: %s", grpAddress)
	}
	dstIP := net.ParseIP(dstAddress)
	if dstIP == nil {
		return fmt.Errorf("invalid destination address: %s", dstAddress)
	}

	cfg := igmp.QuerierConfig{
		Interface:       ifaceName,
		Interval:        time.Duration(interval) * time.Second,
		TTL:             ttl,
		GroupAddress:    grpIP,
		DestAddress:     dstIP,
		MaxResponseTime: time.Duration(maxRespTime) * 100 * time.Millisecond,
		Version:         igmpVersion,
		Logger:          logger,
		RouterAlert:     igmp.Bool(!noRouterAlert),
	}

	querier, err := igmp.NewQuerier(cfg)
	if err != nil {
		return fmt.Errorf("failed to create querier: %w", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	return querier.Run(ctx)
}

// flagStringWithAlias returns the value of the primary flag, falling back to
// the alias if the primary was not explicitly set.
func flagStringWithAlias(flags *pflag.FlagSet, primary, alias string) string {
	if flags.Changed(primary) {
		v, _ := flags.GetString(primary)
		return v
	}
	if flags.Changed(alias) {
		v, _ := flags.GetString(alias)
		return v
	}
	v, _ := flags.GetString(primary)
	return v
}

// flagIntWithAlias returns the value of the primary flag, falling back to
// the alias if the primary was not explicitly set.
func flagIntWithAlias(flags *pflag.FlagSet, primary, alias string) int {
	if flags.Changed(primary) {
		v, _ := flags.GetInt(primary)
		return v
	}
	if flags.Changed(alias) {
		v, _ := flags.GetInt(alias)
		return v
	}
	v, _ := flags.GetInt(primary)
	return v
}

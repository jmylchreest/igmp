// SPDX-License-Identifier: MIT

package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestVersionCommand(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"version"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("version command failed: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Version:") {
		t.Error("version output missing 'Version:' field")
	}
	if !strings.Contains(out, "Commit:") {
		t.Error("version output missing 'Commit:' field")
	}
	if !strings.Contains(out, "Platform:") {
		t.Error("version output missing 'Platform:' field")
	}
}

func TestRootCommandHelp(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"--help"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("root --help failed: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "igmpqd") {
		t.Error("help output should mention igmpqd")
	}
	if !strings.Contains(out, "run") {
		t.Error("help output should list 'run' subcommand")
	}
	if !strings.Contains(out, "version") {
		t.Error("help output should list 'version' subcommand")
	}
}

func TestRunCommandFlags(t *testing.T) {
	// Verify that all expected flags are registered on the run command.
	flags := runCmd.Flags()

	expectedFlags := []string{
		"interface",
		"grp-address",
		"dst-address",
		"interval",
		"ttl",
		"max-response-time",
		"version",
		"debug",
		"no-router-alert",
	}

	for _, name := range expectedFlags {
		f := flags.Lookup(name)
		if f == nil {
			t.Errorf("run command missing expected flag: --%s", name)
		}
	}

	// Verify hidden aliases exist.
	hiddenAliases := []string{
		"grpAddress",
		"dstAddress",
		"maxResponseTime",
	}

	for _, name := range hiddenAliases {
		f := flags.Lookup(name)
		if f == nil {
			t.Errorf("run command missing hidden alias: --%s", name)
			continue
		}
		if !f.Hidden {
			t.Errorf("flag --%s should be hidden", name)
		}
	}
}

func TestFlagStringWithAlias(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected string
	}{
		{
			name:     "primary set",
			args:     []string{"--primary", "hello"},
			expected: "hello",
		},
		{
			name:     "alias set",
			args:     []string{"--alias", "world"},
			expected: "world",
		},
		{
			name:     "neither set uses default",
			args:     nil,
			expected: "default",
		},
		{
			name:     "both set primary wins",
			args:     []string{"--primary", "from-primary", "--alias", "from-alias"},
			expected: "from-primary",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{Use: "test", Run: func(*cobra.Command, []string) {}}
			cmd.Flags().String("primary", "default", "primary flag")
			cmd.Flags().String("alias", "default", "alias flag")

			if tt.args != nil {
				if err := cmd.Flags().Parse(tt.args); err != nil {
					t.Fatalf("parse flags: %v", err)
				}
			}

			got := flagStringWithAlias(cmd.Flags(), "primary", "alias")
			if got != tt.expected {
				t.Errorf("flagStringWithAlias = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestFlagIntWithAlias(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected int
	}{
		{
			name:     "primary set",
			args:     []string{"--primary", "42"},
			expected: 42,
		},
		{
			name:     "alias set",
			args:     []string{"--alias", "99"},
			expected: 99,
		},
		{
			name:     "neither set uses default",
			args:     nil,
			expected: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{Use: "test", Run: func(*cobra.Command, []string) {}}
			cmd.Flags().Int("primary", 10, "primary flag")
			cmd.Flags().Int("alias", 10, "alias flag")

			if tt.args != nil {
				if err := cmd.Flags().Parse(tt.args); err != nil {
					t.Fatalf("parse flags: %v", err)
				}
			}

			got := flagIntWithAlias(cmd.Flags(), "primary", "alias")
			if got != tt.expected {
				t.Errorf("flagIntWithAlias = %d, want %d", got, tt.expected)
			}
		})
	}
}

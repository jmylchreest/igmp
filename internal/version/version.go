// SPDX-License-Identifier: MIT

// Package version provides build-time version information shared across
// CLI tools in this project. Variables are populated via -ldflags at build time.
package version

import (
	"fmt"
	"runtime"
	"time"
)

// These variables are set at build time via -ldflags.
//
//	go build -ldflags "-X github.com/jmylchreest/igmp/internal/version.GitCommit=abc123
//	  -X github.com/jmylchreest/igmp/internal/version.GitDescribe=v1.0.0
//	  -X github.com/jmylchreest/igmp/internal/version.BuildTime=1709827200"
var (
	GitCommit   string = "unknown"
	GitDescribe string = "dev"
	BuildTime   string = "0"
)

// Info returns a formatted version string.
func Info() string {
	bt := "unknown"
	if BuildTime != "0" && BuildTime != "" {
		if ts, err := time.Parse(time.RFC3339, BuildTime); err == nil {
			bt = ts.Format(time.RFC1123Z)
		} else {
			bt = BuildTime
		}
	}

	return fmt.Sprintf("Version:    %s\nCommit:     %s\nBuilt:      %s\nGo:         %s\nPlatform:   %s/%s",
		GitDescribe, GitCommit, bt, runtime.Version(), runtime.GOOS, runtime.GOARCH)
}

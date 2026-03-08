// SPDX-License-Identifier: MIT

package version

import (
	"runtime"
	"strings"
	"testing"
)

func TestInfoDefaults(t *testing.T) {
	// With default values, Info should return sensible output.
	info := Info()

	if !strings.Contains(info, "Version:") {
		t.Error("Info() missing Version field")
	}
	if !strings.Contains(info, "dev") {
		t.Error("Info() should show default version 'dev'")
	}
	if !strings.Contains(info, "unknown") {
		t.Error("Info() should show 'unknown' for default commit and build time")
	}
	if !strings.Contains(info, runtime.Version()) {
		t.Errorf("Info() should include Go version %s", runtime.Version())
	}
	if !strings.Contains(info, runtime.GOOS+"/"+runtime.GOARCH) {
		t.Error("Info() should include platform")
	}
}

func TestInfoCustomValues(t *testing.T) {
	// Save originals and restore after test.
	origCommit := GitCommit
	origDescribe := GitDescribe
	origBuild := BuildTime
	t.Cleanup(func() {
		GitCommit = origCommit
		GitDescribe = origDescribe
		BuildTime = origBuild
	})

	GitCommit = "abc123def456"
	GitDescribe = "v2.0.0"
	BuildTime = "2025-01-15T10:30:00Z"

	info := Info()

	if !strings.Contains(info, "v2.0.0") {
		t.Error("Info() should show custom version")
	}
	if !strings.Contains(info, "abc123def456") {
		t.Error("Info() should show custom commit")
	}
	// BuildTime should be parsed and reformatted.
	if strings.Contains(info, "unknown") {
		t.Error("Info() should not show 'unknown' when BuildTime is valid")
	}
}

func TestInfoNonRFC3339BuildTime(t *testing.T) {
	origBuild := BuildTime
	t.Cleanup(func() { BuildTime = origBuild })

	// Non-RFC3339 string should be used as-is.
	BuildTime = "2025-01-15 by CI"
	info := Info()

	if !strings.Contains(info, "2025-01-15 by CI") {
		t.Error("Info() should pass through non-RFC3339 BuildTime as-is")
	}
}

func TestInfoEmptyBuildTime(t *testing.T) {
	origBuild := BuildTime
	t.Cleanup(func() { BuildTime = origBuild })

	BuildTime = ""
	info := Info()

	if !strings.Contains(info, "Built:      unknown") {
		t.Error("Info() should show 'unknown' for empty BuildTime")
	}
}

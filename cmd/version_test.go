package cmd

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
)

func TestVersionCommand(t *testing.T) {
	// Set dummy values for version variables in the cmd package
	originalGitCommit := GitCommit
	originalGitDescribe := GitDescribe
	originalBuildTime := BuildTime

	GitCommit = "testcommit123"
	GitDescribe = "v0.1.0-test"
	testTime := time.Date(2023, 1, 15, 10, 30, 0, 0, time.UTC)
	BuildTime = testTime.Unix()

	defer func() {
		GitCommit = originalGitCommit
		GitDescribe = originalGitDescribe
		BuildTime = originalBuildTime
	}()

	// Create a new RootCmd for this test to ensure isolation
	testRootCmd := &cobra.Command{Use: "igmpqd_test_version"}
	testRootCmd.AddCommand(versionCmd) // versionCmd is global in cmd package

	actual := new(bytes.Buffer)
	testRootCmd.SetOut(actual)
	testRootCmd.SetErr(actual)
	testRootCmd.SetArgs([]string{"version"})

	err := testRootCmd.Execute()
	if err != nil {
		t.Fatalf("version command execution failed: %v", err)
	}

	output := actual.String()

	expectedVersionString := "Version:\tv0.1.0-test (Commit: testcommit123)"
	if !strings.Contains(output, expectedVersionString) {
		t.Errorf("Output does not contain expected version string.\nExpected to contain: %s\nGot:\n%s", expectedVersionString, output)
	}

	expectedBuildTimeString := testTime.Format(time.RFC1123Z)
	expectedBuiltString := "Built:\t\t" + expectedBuildTimeString
	if !strings.Contains(output, expectedBuiltString) {
		t.Errorf("Output does not contain expected build time string.\nExpected to contain: %s\nGot:\n%s", expectedBuiltString, output)
	}

	expectedFingerprintString := "Fingerprint:"
	if !strings.Contains(output, expectedFingerprintString) {
		t.Errorf("Output does not contain expected Fingerprint string.\nExpected to contain: %s\nGot:\n%s", expectedFingerprintString, output)
	}
}

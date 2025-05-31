package cmd

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

// createNewRootCmdForTest creates a new RootCmd and adds a fresh run command to it.
// This helps in isolating tests.
func createNewRootCmdWithRunCmdForTest() *cobra.Command {
	newRootCmd := &cobra.Command{Use: "igmpqd_test"} // Use a different name to avoid conflicts if any global RootCmd is used

	// Re-create versionCmd and add it, as RootCmd is new
	// (or ensure versionCmd is also created via a constructor if it also needs testing context)
	// For now, assume versionCmd from cmd package is okay to add if needed, or skip if not essential for run tests.
	// If version tests are failing, it might be because RootCmd is being meddled with.
	// Let's ensure versionCmd is added to this test-local RootCmd if Execute is called on it.
	newRootCmd.AddCommand(versionCmd) // versionCmd is global in cmd package

	runCmdToTest := &cobra.Command{
		Use: "run",
		Short: "Test Run Command",
		Run: func(cmd *cobra.Command, args []string) {
			// Test Run, can be empty for flag testing
		},
	}
	SetupRunCommand(runCmdToTest) // Use the setup function from cmd/run.go
	newRootCmd.AddCommand(runCmdToTest)
	return newRootCmd
}

func TestRunCommand_DefaultValues(t *testing.T) {
	viper.Reset() // Reset Viper for each test
	testRootCmd := createNewRootCmdWithRunCmdForTest()

	testRootCmd.SetArgs([]string{"run"}) // Execute the run command with no args
	err := testRootCmd.Execute()
	assert.NoError(t, err)

	// Check Viper values (defaults)
	assert.Equal(t, "224.0.0.1", viper.GetString("dstAddress"), "Default DstAddress")
	assert.Equal(t, "", viper.GetString("interface"), "Default Interface")
	assert.Equal(t, 30, viper.GetInt("interval"), "Default Interval")
	assert.Equal(t, 1, viper.GetInt("ttl"), "Default TTL")
	assert.Equal(t, 100, viper.GetInt("maxResponseTime"), "Default MaxResponseTime")
	assert.Equal(t, "0.0.0.0", viper.GetString("grpAddress"), "Default GrpAddress")
	assert.False(t, viper.GetBool("debug"), "Default Debug")
}

func TestRunCommand_SetDstAddressFlag(t *testing.T) {
	viper.Reset()
	testRootCmd := createNewRootCmdWithRunCmdForTest()

	testRootCmd.SetArgs([]string{"run", "--dstAddress", "1.2.3.4"})
	err := testRootCmd.Execute()
	assert.NoError(t, err)
	assert.Equal(t, "1.2.3.4", viper.GetString("dstAddress"))
}

func TestRunCommand_SetInterfaceFlag(t *testing.T) {
	viper.Reset()
	testRootCmd := createNewRootCmdWithRunCmdForTest()

	testRootCmd.SetArgs([]string{"run", "-I", "ethTest"})
	err := testRootCmd.Execute()
	assert.NoError(t, err)
	assert.Equal(t, "ethTest", viper.GetString("interface"))
}

func TestRunCommand_SetIntervalFlag(t *testing.T) {
	viper.Reset()
	testRootCmd := createNewRootCmdWithRunCmdForTest()

	testRootCmd.SetArgs([]string{"run", "--interval", "60"})
	err := testRootCmd.Execute()
	assert.NoError(t, err)
	assert.Equal(t, 60, viper.GetInt("interval"))
}

func TestRunCommand_SetTTLFlag(t *testing.T) {
	viper.Reset()
	testRootCmd := createNewRootCmdWithRunCmdForTest()

	testRootCmd.SetArgs([]string{"run", "-t", "5"})
	err := testRootCmd.Execute()
	assert.NoError(t, err)
	assert.Equal(t, 5, viper.GetInt("ttl"))
}

func TestRunCommand_SetMaxResponseTimeFlag(t *testing.T) {
	viper.Reset()
	testRootCmd := createNewRootCmdWithRunCmdForTest()

	testRootCmd.SetArgs([]string{"run", "-m", "50"})
	err := testRootCmd.Execute()
	assert.NoError(t, err)
	assert.Equal(t, 50, viper.GetInt("maxResponseTime"))
}

func TestRunCommand_SetGrpAddressFlag(t *testing.T) {
	viper.Reset()
	testRootCmd := createNewRootCmdWithRunCmdForTest()

	testRootCmd.SetArgs([]string{"run", "--grpAddress", "239.0.0.1"})
	err := testRootCmd.Execute()
	assert.NoError(t, err)
	assert.Equal(t, "239.0.0.1", viper.GetString("grpAddress"))
}

func TestRunCommand_SetDebugFlag(t *testing.T) {
	viper.Reset()
	testRootCmd := createNewRootCmdWithRunCmdForTest()

	// Need to capture output to see if debug logs are generated.
	// For now, just test if viper gets the bool.
	// The debug() function itself checks viper.GetBool("debug").
	testRootCmd.SetArgs([]string{"run", "--debug"})
	err := testRootCmd.Execute()
	assert.NoError(t, err)
	assert.True(t, viper.GetBool("debug"))
}

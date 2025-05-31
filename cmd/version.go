package cmd

import (
	"runtime"
	// "strconv" // BuildTime will be int64 in cmd package
	"time"

	"github.com/spf13/cobra"
	// No import of main needed here
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show the current version of igmpqd",
	Run: func(cmdInstance *cobra.Command, args []string) { // Renamed to cmdInstance
		var builtTimeDisplay string
		if BuildTime != 0 {
			builtTimeDisplay = time.Unix(BuildTime, 0).Format(time.RFC1123Z)
		} else {
			builtTimeDisplay = "N/A (BuildTime not available or parse error)"
		}

		// Use cmdInstance.Printf or fmt.Fprintln(cmdInstance.OutOrStdout(), ...) for output
		cmdInstance.Printf("Version:\t%s (Commit: %s)\n", GitDescribe, GitCommit)
		cmdInstance.Printf("Built:\t\t%s\n", builtTimeDisplay)
		cmdInstance.Printf("Fingerprint:\t%s/%s/%s/%s\n", runtime.Compiler, runtime.GOOS, runtime.GOARCH, runtime.Version())
	},
}

func init() {
	RootCmd.AddCommand(versionCmd)
}

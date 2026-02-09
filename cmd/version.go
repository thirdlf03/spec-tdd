package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

var (
	// Version is the version of the application
	// This will be set by the build process using ldflags
	Version = "dev"

	// Commit is the git commit hash
	// This will be set by the build process using ldflags
	Commit = "none"

	// BuildDate is the date the binary was built
	// This will be set by the build process using ldflags
	BuildDate = "unknown"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number",
	Long:  `All software has versions. This is the version of this application.`,
	Run: func(cmd *cobra.Command, args []string) {
		out := cmd.OutOrStdout()
		fmt.Fprintf(out, "Version:    %s\n", Version)
		fmt.Fprintf(out, "Commit:     %s\n", Commit)
		fmt.Fprintf(out, "Build Date: %s\n", BuildDate)
		fmt.Fprintf(out, "Go Version: %s\n", runtime.Version())
		fmt.Fprintf(out, "Platform:   %s/%s\n", runtime.GOOS, runtime.GOARCH)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

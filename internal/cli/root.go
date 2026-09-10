// Package cli wires the vitrine subcommands.
package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Version is set at build time via -ldflags.
var Version = "dev"

func newRoot() *cobra.Command {
	root := &cobra.Command{
		Use:           "vitrine",
		Short:         "Run coding agents with read-only access to everything",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print the version",
		Run: func(cmd *cobra.Command, _ []string) {
			fmt.Fprintln(cmd.OutOrStdout(), Version)
		},
	})
	return root
}

// Execute runs the CLI with args (excluding argv[0]) and returns the exit code.
func Execute(args []string) int {
	root := newRoot()
	root.SetArgs(args)
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "vitrine:", err)
		return 1
	}
	return 0
}

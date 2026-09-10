// Package cli wires the vitrine subcommands.
package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Version is set at build time via -ldflags.
var Version = "dev"

// exitError carries a specific exit code out of a RunE.
type exitError struct{ code int }

func (e exitError) Error() string { return fmt.Sprintf("exit %d", e.code) }

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
	root.AddCommand(newProbeCmd())
	root.AddCommand(platformCommands()...)
	return root
}

// Execute runs the CLI with args (excluding argv[0]) and returns the exit code.
func Execute(args []string) int {
	root := newRoot()
	root.SetArgs(args)
	if err := root.Execute(); err != nil {
		var ee exitError
		if errors.As(err, &ee) {
			return ee.code
		}
		fmt.Fprintln(os.Stderr, "vitrine:", err)
		return 1
	}
	return 0
}

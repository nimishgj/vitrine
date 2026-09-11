// Package cli wires the vitrine subcommands.
package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/nimishgj/vitrine/internal/profile"
)

// Version is set at build time via -ldflags.
var Version = "dev"

// exitError carries a specific exit code out of a RunE.
type exitError struct{ code int }

func (e exitError) Error() string { return fmt.Sprintf("exit %d", e.code) }

func newRoot(e *env) *cobra.Command {
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
	root.AddCommand(newInitCmd(e))
	root.AddCommand(newRunCmd(e))
	root.AddCommand(newGrantCmd(e), newRevokeCmd(e), newGrantsCmd(e))
	root.AddCommand(newTargetCmd(e))
	root.AddCommand(newStatusCmd(e), newAuditCmd(e))
	root.AddCommand(newProbeCmd())
	root.AddCommand(platformCommands()...)
	return root
}

// Execute runs the CLI. When argv0 is one of the agent shims, the whole
// invocation is a sandboxed launch of that agent.
func Execute(argv0 string, args []string) int {
	e, err := loadEnv()
	if err != nil {
		fmt.Fprintln(os.Stderr, "vitrine:", err)
		return 1
	}
	if name, ok := profile.FromArgv0(argv0); ok {
		code, err := runSession(e, name, args)
		if err != nil {
			fmt.Fprintln(os.Stderr, "vitrine:", err)
			if code == 0 {
				code = 1
			}
		}
		return code
	}
	root := newRoot(e)
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

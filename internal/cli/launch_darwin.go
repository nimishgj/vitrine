//go:build darwin

package cli

import (
	"github.com/spf13/cobra"

	"github.com/nimishgj/vitrine/internal/sandbox/darwin"
)

func newLaunchCmd() *cobra.Command {
	var check bool
	var adopt string
	cmd := &cobra.Command{
		Use:    "launch SPEC_FILE -- CMD [ARGS...]",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if check {
				return nil // reachable only if the sudoers rule works
			}
			if adopt != "" {
				return darwin.Adopt(adopt)
			}
			if len(args) < 2 {
				return cmd.Usage()
			}
			return darwin.Launch(args[0], args[1:])
		},
	}
	cmd.Flags().BoolVar(&check, "check", false, "")
	cmd.Flags().StringVar(&adopt, "adopt", "", "")
	return cmd
}

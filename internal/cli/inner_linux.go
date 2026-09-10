//go:build linux

package cli

import (
	"fmt"
	"os"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/nimishgj/vitrine/internal/sandbox"
	lnx "github.com/nimishgj/vitrine/internal/sandbox/linux"
)

func newInnerCmd() *cobra.Command {
	var specFD int
	cmd := &cobra.Command{
		Use:    "_inner -- CMD [ARGS...]",
		Hidden: true,
		Args:   cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			f := os.NewFile(uintptr(specFD), "spec")
			spec, err := sandbox.ReadSpec(f)
			f.Close()
			if err != nil {
				return fmt.Errorf("read spec: %w", err)
			}
			if err := lnx.ApplyLandlock(spec); err != nil {
				// Landlock is the second layer; the namespace is primary. Warn only.
				fmt.Fprintln(os.Stderr, "vitrine: landlock not applied:", err)
			}
			return syscall.Exec(args[0], args, os.Environ())
		},
	}
	cmd.Flags().IntVar(&specFD, "spec-fd", 4, "")
	return cmd
}

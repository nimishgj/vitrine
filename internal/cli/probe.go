package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/nimishgj/vitrine/internal/probe"
)

func newProbeCmd() *cobra.Command {
	var p probe.Params
	var out string
	cmd := &cobra.Command{
		Use:    "probe",
		Short:  "Internal: run isolation checks from inside a sandbox",
		Hidden: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cs := probe.Run(p)
			b, _ := json.Marshal(cs)
			if out != "" {
				if err := os.WriteFile(out, b, 0o600); err != nil {
					return err
				}
			} else {
				fmt.Fprintln(cmd.OutOrStdout(), string(b))
			}
			if !probe.Passed(cs) {
				return exitError{code: 2}
			}
			return nil
		},
	}
	f := cmd.Flags()
	f.StringVar(&p.CredPath, "cred-path", "", "")
	f.StringVar(&p.HomeDir, "home", "", "")
	f.StringVar(&p.StateFile, "state-file", "", "")
	f.StringVar(&p.OutsidePath, "outside", "", "")
	f.StringVar(&p.WriteGrant, "write-grant", "", "")
	f.StringVar(&p.ReadGrant, "read-grant", "", "")
	f.StringVar(&p.AgentHome, "agent-home", "", "")
	f.StringVar(&p.Scratch, "scratch", "", "")
	f.StringVar(&p.SymlinkInGrant, "symlink", "", "")
	f.StringVar(&p.CanaryEnv, "canary-env", "VITRINE_CANARY", "")
	f.StringVar(&p.NetAddr, "net", "", "")
	f.StringVar(&out, "out", "", "write JSON results to this file instead of stdout")
	return cmd
}

package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/nimishgj/vitrine/internal/audit"
	"github.com/nimishgj/vitrine/internal/config"
	"github.com/nimishgj/vitrine/internal/doctor"
	"github.com/nimishgj/vitrine/internal/grants"
	"github.com/nimishgj/vitrine/internal/paths"
	"github.com/nimishgj/vitrine/internal/probe"
	"github.com/nimishgj/vitrine/internal/sysuser"
)

func newDoctorCmd(e *env) *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Prove the sandbox works on this machine",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config.Load(e.Layout.Config)
			if err != nil {
				return err
			}
			b, err := backendFactory(cfg.Backend, e.Home)
			if err != nil {
				return err
			}
			agentHome := func(n string) string { return paths.AgentHome(e.Layout, n) }
			apply := func(grants.Grant) error { return nil }
			remove := func(grants.Grant) error { return nil }
			if e.GOOS == "darwin" {
				agentHome = sysuser.AgentHome
				apply = func(g grants.Grant) error { return grants.ApplyACL(g, e.Home, e.Engineer) }
				remove = func(g grants.Grant) error { return grants.RemoveACL(g, e.Home, nil) }
			}
			fmt.Fprintf(e.Stdout, "backend: %s\n", b.Name())
			cs, berr, err := doctor.Run(context.Background(), doctor.Deps{
				Home: e.Home, Engineer: e.Engineer, GOOS: e.GOOS, VitrineBin: e.Exe, Backend: b,
				AgentHome: agentHome, ApplyGrant: apply, RemoveGrant: remove,
			})
			if err != nil {
				return err
			}
			if berr != nil {
				fmt.Fprintf(e.Stdout, "available: no\n  %v\n", berr)
				e.Log.Append(audit.TypeDoctor, map[string]any{"backend": b.Name(), "passed": false, "error": berr.Error()})
				return exitError{code: 1}
			}
			fmt.Fprint(e.Stdout, probe.Format(cs))
			passed := probe.Passed(cs)
			e.Log.Append(audit.TypeDoctor, map[string]any{"backend": b.Name(), "passed": passed, "checks": cs})
			if !passed {
				fmt.Fprintln(e.Stdout, "\nFAILED: at least one operation that must be denied was allowed. Do not run agents until this passes.")
				return exitError{code: 2}
			}
			fmt.Fprintln(e.Stdout, "\nall checks passed")
			return nil
		},
	}
}

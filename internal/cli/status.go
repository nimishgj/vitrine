package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/nimishgj/vitrine/internal/audit"
	"github.com/nimishgj/vitrine/internal/config"
)

func newStatusCmd(e *env) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show backend, enforcement level, grants and targets",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config.Load(e.Layout.Config)
			if err != nil {
				return err
			}
			b, err := backendFactory(cfg.Backend, e.Home)
			if err != nil {
				return err
			}
			fmt.Fprintf(e.Stdout, "vitrine %s\nstate: %s\nbackend: %s\n", Version, e.Layout.Root, b.Name())
			if err := b.Available(); err != nil {
				fmt.Fprintf(e.Stdout, "available: no (%v)\n", err)
			} else {
				fmt.Fprintln(e.Stdout, "available: yes")
			}
			fmt.Fprintln(e.Stdout, "enforcement: unmanaged (nothing prevents running agent binaries outside vitrine on this machine)")

			rows, err := listGrants(e)
			if err != nil {
				return err
			}
			if len(rows) == 0 {
				fmt.Fprintln(e.Stdout, "grants: none")
			} else {
				fmt.Fprintln(e.Stdout, "grants:")
				for _, r := range rows {
					flag := ""
					if !r.Recognised {
						flag = "  UNRECOGNISED"
					}
					fmt.Fprintf(e.Stdout, "  %-6s %s%s\n", r.Grant.Mode, r.Grant.Path, flag)
				}
			}
			if len(cfg.Targets) == 0 {
				fmt.Fprintln(e.Stdout, "targets: none")
			} else {
				fmt.Fprintln(e.Stdout, "targets:")
				for _, t := range cfg.Targets {
					fmt.Fprintf(e.Stdout, "  %-20s %s\n", t.Name, t.Kind)
				}
			}
			evs, err := e.Log.ReadAll()
			if err != nil {
				return err
			}
			doctor := "never run"
			for _, ev := range evs {
				if ev.Type == audit.TypeDoctor {
					if ok, _ := ev.Data["passed"].(bool); ok {
						doctor = "passed at " + ev.TS.Format("2006-01-02 15:04")
					} else {
						doctor = "FAILED at " + ev.TS.Format("2006-01-02 15:04")
					}
				}
			}
			fmt.Fprintf(e.Stdout, "doctor: %s\n", doctor)
			return nil
		},
	}
}

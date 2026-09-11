package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/nimishgj/vitrine/internal/config"
	"github.com/nimishgj/vitrine/internal/provider"
)

func newTargetCmd(e *env) *cobra.Command {
	cmd := &cobra.Command{Use: "target", Short: "Manage remote targets"}

	var kind string
	var sets []string
	add := &cobra.Command{
		Use:   "add NAME --kind KIND [--set key=value ...]",
		Short: "Add a target",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if _, ok := provider.Lookup(kind); !ok {
				return fmt.Errorf("unknown target kind %q; available: %s", kind, strings.Join(provider.Kinds(), ", "))
			}
			settings := map[string]string{}
			for _, kv := range sets {
				k, v, ok := strings.Cut(kv, "=")
				if !ok {
					return fmt.Errorf("--set expects key=value, got %q", kv)
				}
				settings[k] = v
			}
			cfg, err := config.Load(e.Layout.Config)
			if err != nil {
				return err
			}
			if err := cfg.AddTarget(config.Target{Name: args[0], Kind: kind, Settings: settings}); err != nil {
				return err
			}
			if err := config.Save(e.Layout.Config, cfg, e.Log); err != nil {
				return err
			}
			fmt.Fprintf(e.Stdout, "added target %s (%s)\n", args[0], kind)
			return nil
		},
	}
	add.Flags().StringVar(&kind, "kind", "", "provider kind")
	add.Flags().StringArrayVar(&sets, "set", nil, "provider setting key=value")
	add.MarkFlagRequired("kind")

	list := &cobra.Command{
		Use:   "list",
		Short: "List targets",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config.Load(e.Layout.Config)
			if err != nil {
				return err
			}
			if len(cfg.Targets) == 0 {
				fmt.Fprintln(e.Stdout, "no targets")
				return nil
			}
			for _, t := range cfg.Targets {
				var kv []string
				for k, v := range t.Settings {
					kv = append(kv, k+"="+v)
				}
				fmt.Fprintf(e.Stdout, "%-20s %-12s %s\n", t.Name, t.Kind, strings.Join(kv, " "))
			}
			return nil
		},
	}

	remove := &cobra.Command{
		Use:   "remove NAME",
		Short: "Remove a target",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(e.Layout.Config)
			if err != nil {
				return err
			}
			if !cfg.RemoveTarget(args[0]) {
				return fmt.Errorf("no target named %q", args[0])
			}
			if err := config.Save(e.Layout.Config, cfg, e.Log); err != nil {
				return err
			}
			fmt.Fprintf(e.Stdout, "removed target %s\n", args[0])
			return nil
		},
	}

	cmd.AddCommand(add, list, remove)
	return cmd
}

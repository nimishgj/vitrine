package cli

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/nimishgj/vitrine/internal/audit"
)

func newAuditCmd(e *env) *cobra.Command {
	var asJSON bool
	var since time.Duration
	cmd := &cobra.Command{
		Use:   "audit [--since DURATION] [--json]",
		Short: "Show the audit log",
		RunE: func(cmd *cobra.Command, _ []string) error {
			evs, err := e.Log.ReadAll()
			if err != nil {
				return err
			}
			cutoff := time.Time{}
			if since > 0 {
				cutoff = time.Now().Add(-since)
			}
			for _, ev := range evs {
				if ev.TS.Before(cutoff) {
					continue
				}
				if asJSON {
					b, _ := json.Marshal(ev)
					fmt.Fprintln(e.Stdout, string(b))
					continue
				}
				fmt.Fprintf(e.Stdout, "%s  %-14s %s\n", ev.TS.Local().Format("2006-01-02 15:04:05"), ev.Type, summary(ev))
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "print raw JSON lines")
	cmd.Flags().DurationVar(&since, "since", 0, "only events newer than this (e.g. 24h)")
	cmd.AddCommand(&cobra.Command{
		Use:   "verify",
		Short: "Verify the audit chain",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := e.Log.Verify(); err != nil {
				fmt.Fprintf(e.Stderr, "vitrine: %v\n", err)
				return exitError{code: 3}
			}
			evs, _ := e.Log.ReadAll()
			fmt.Fprintf(e.Stdout, "audit chain intact (%d events)\n", len(evs))
			return nil
		},
	})
	return cmd
}

func summary(ev audit.Event) string {
	d := ev.Data
	switch ev.Type {
	case audit.TypeSessionStart:
		return fmt.Sprintf("%v %v in %v (%v grant)", d["session"], d["agent"], d["cwd"], d["grant_mode"])
	case audit.TypeSessionEnd:
		return fmt.Sprintf("%v exit %v", d["session"], d["exit_code"])
	case audit.TypeGrantAdd, audit.TypeGrantRevoke:
		return fmt.Sprintf("%v %v", d["mode"], d["path"])
	case audit.TypeGrantAccept:
		return fmt.Sprintf("%v grants accepted", d["count"])
	case audit.TypeTamper:
		return fmt.Sprintf("%v modified outside vitrine (mtime %v)", d["file"], d["mtime"])
	case audit.TypeDoctor:
		if ok, _ := d["passed"].(bool); ok {
			return "passed"
		}
		return "FAILED"
	default:
		var parts []string
		for k, v := range d {
			parts = append(parts, fmt.Sprintf("%s=%v", k, v))
		}
		return strings.Join(parts, " ")
	}
}

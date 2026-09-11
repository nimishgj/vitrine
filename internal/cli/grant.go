package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/nimishgj/vitrine/internal/audit"
	"github.com/nimishgj/vitrine/internal/grants"
	"github.com/nimishgj/vitrine/internal/sandbox"
)

func newGrantCmd(e *env) *cobra.Command {
	var readOnly bool
	cmd := &cobra.Command{
		Use:   "grant PATH [--read-only]",
		Short: "Allow agents to write (or only read) under PATH",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := grants.Canonical(args[0])
			if err != nil {
				return err
			}
			mode := grants.ModeWrite
			if readOnly {
				mode = grants.ModeRead
			}
			if err := addGrant(e, grants.Grant{Path: p, Mode: mode, CreatedAt: time.Now().UTC(), Source: "cli"}); err != nil {
				return err
			}
			fmt.Fprintf(e.Stdout, "granted %s on %s\n", mode, p)
			return nil
		},
	}
	cmd.Flags().BoolVar(&readOnly, "read-only", false, "grant read access only")
	return cmd
}

func newRevokeCmd(e *env) *cobra.Command {
	return &cobra.Command{
		Use:   "revoke PATH",
		Short: "Remove a grant",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := grants.Canonical(args[0])
			if err != nil {
				return err
			}
			return revokeGrant(e, p)
		},
	}
}

func revokeGrant(e *env, p string) error {
	s, err := grants.Load(e.Layout.Grants)
	if err != nil {
		return err
	}
	g, ok := s.Match(p)
	if !ok || g.Path != p {
		return fmt.Errorf("no grant exactly at %s", p)
	}
	s.Remove(p)
	if err := grants.Save(e.Layout.Grants, s); err != nil {
		return err
	}
	if e.GOOS == "darwin" {
		if err := grants.RemoveACL(g, e.Home, s.Grants); err != nil {
			return err
		}
	}
	h, err := grants.FileHash(e.Layout.Grants)
	if err != nil {
		return err
	}
	_, err = e.Log.Append(audit.TypeGrantRevoke, map[string]any{
		"path": p, "mode": string(g.Mode), "grants_hash": h, "grants": audit.GrantSnapshot(s),
	})
	if err == nil {
		fmt.Fprintf(e.Stdout, "revoked %s\n", p)
	}
	return err
}

type grantRow struct {
	Grant      grants.Grant
	Recognised bool
}

// listGrants returns current grants flagged by whether the chain recognises them.
func listGrants(e *env) ([]grantRow, error) {
	s, err := grants.Load(e.Layout.Grants)
	if err != nil {
		return nil, err
	}
	evs, err := e.Log.ReadAll()
	if err != nil {
		return nil, err
	}
	expected, _ := audit.ExpectedHashes(evs)
	actual, err := grants.FileHash(e.Layout.Grants)
	if err != nil {
		return nil, err
	}
	rec := map[string]grants.Grant{}
	for _, g := range audit.RecognisedGrants(evs) {
		rec[g.Path] = g
	}
	var rows []grantRow
	for _, g := range s.Grants {
		r, ok := rec[g.Path]
		rows = append(rows, grantRow{Grant: g, Recognised: expected == actual || (ok && r.Mode == g.Mode)})
	}
	return rows, nil
}

func newGrantsCmd(e *env) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "grants",
		Short: "List grants",
		RunE: func(cmd *cobra.Command, _ []string) error {
			rows, err := listGrants(e)
			if err != nil {
				return err
			}
			if len(rows) == 0 {
				fmt.Fprintln(e.Stdout, "no grants")
				return nil
			}
			for _, r := range rows {
				flag := ""
				if !r.Recognised {
					flag = "  UNRECOGNISED (run: vitrine grants accept)"
				}
				fmt.Fprintf(e.Stdout, "%-6s %-7s %s  %s%s\n", r.Grant.Mode, r.Grant.Source, r.Grant.CreatedAt.Format("2006-01-02"), r.Grant.Path, flag)
			}
			return nil
		},
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "accept",
		Short: "Adopt grants that were added outside vitrine",
		RunE:  func(cmd *cobra.Command, _ []string) error { return acceptGrants(e) },
	})
	return cmd
}

// acceptGrants re-validates every entry in the file, applies ACLs, and
// records the whole set as recognised.
func acceptGrants(e *env) error {
	s, err := grants.Load(e.Layout.Grants)
	if err != nil {
		return err
	}
	roots := sandbox.SystemRoots(e.GOOS, nil)
	for i := range s.Grants {
		g := &s.Grants[i]
		p, err := grants.Canonical(g.Path)
		if err != nil {
			return err
		}
		g.Path = p
		if err := grants.Refuse(p, e.Home, roots); err != nil {
			return err
		}
		if g.CreatedAt.IsZero() {
			g.CreatedAt = time.Now().UTC()
		}
		if g.Source == "" {
			g.Source = "accepted"
		}
		if e.GOOS == "darwin" {
			if err := grants.ApplyACL(*g, e.Home, e.Engineer); err != nil {
				return err
			}
		}
	}
	if err := grants.Save(e.Layout.Grants, s); err != nil {
		return err
	}
	h, err := grants.FileHash(e.Layout.Grants)
	if err != nil {
		return err
	}
	_, err = e.Log.Append(audit.TypeGrantAccept, map[string]any{
		"count": len(s.Grants), "grants_hash": h, "grants": audit.GrantSnapshot(s),
	})
	if err == nil {
		fmt.Fprintf(e.Stdout, "accepted %d grant(s)\n", len(s.Grants))
	}
	return err
}

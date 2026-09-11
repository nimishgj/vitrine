package cli

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/nimishgj/vitrine/internal/grants"
)

var errNoGrant = errors.New("no grant covers this directory and stdin is not a terminal; run: vitrine grant <path> [--read-only]")

// promptTrust asks once whether the agent may use dir. Returns the chosen
// mode and whether to proceed. Fails closed when not interactive.
func promptTrust(in io.Reader, out io.Writer, isTTY bool, agent, dir string) (grants.Mode, bool, error) {
	if !isTTY {
		return "", false, errNoGrant
	}
	r := bufio.NewReader(in)
	for {
		fmt.Fprintf(out, "vitrine: %s wants to run in %s\n  [w] allow writes here    [r] read-only    [n] cancel\n> ", agent, dir)
		line, err := r.ReadString('\n')
		if err != nil && line == "" {
			return "", false, fmt.Errorf("no answer: %w", err)
		}
		switch strings.ToLower(strings.TrimSpace(line)) {
		case "w":
			return grants.ModeWrite, true, nil
		case "r":
			return grants.ModeRead, true, nil
		case "n":
			return "", false, nil
		}
		fmt.Fprintln(out, "please answer w, r, or n")
	}
}

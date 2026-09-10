//go:build !linux && !darwin

package cli

import "github.com/spf13/cobra"

func platformCommands() []*cobra.Command { return nil }

//go:build darwin

package cli

import "github.com/spf13/cobra"

func platformCommands() []*cobra.Command { return []*cobra.Command{newLaunchCmd()} }

package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "Show the current source, version and managed files",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		cmd.SilenceUsage = true
		s, err := newSyncer()
		if err != nil {
			return err
		}
		m, l, err := s.Status()
		if err != nil {
			return err
		}

		out := cmd.OutOrStdout()
		fmt.Fprintf(out, "source:  %s\n", m.Source)
		fmt.Fprintf(out, "version: %s\n", m.Version)
		fmt.Fprintf(out, "target:  %s\n", m.Target)
		if l == nil {
			fmt.Fprintln(out, "lock:    (not synced yet, run `update`)")
			return nil
		}
		fmt.Fprintf(out, "locked:  %s @ %s\n", l.Version, l.Resolved)
		fmt.Fprintf(out, "managed: %d files\n", len(l.Managed))
		for _, f := range l.Managed {
			fmt.Fprintf(out, "  - %s\n", f)
		}
		return nil
	},
}

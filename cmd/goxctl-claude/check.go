package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Verify local managed files match the lock (for CI)",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		cmd.SilenceUsage = true
		s, err := newSyncer()
		if err != nil {
			return err
		}
		if err := s.Check(); err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), "ok: managed files match lock")
		return nil
	},
}

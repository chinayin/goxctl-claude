package main

import (
	"github.com/chinayin/goxctl-claude/internal/ui"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update [version]",
	Short: "Upgrade to the latest release (no arg) or switch to a given version",
	Long: `Sync the standards in one of two modes:

  no args     upgrade to the latest release.
  <version>   switch to that tag (e.g. v0.2.0).

Both rewrite the manifest and lock; review the .kiro/steering changes via git diff before committing.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		s, err := newSyncer()
		if err != nil {
			return err
		}
		var version string
		if len(args) == 1 {
			version = args[0]
		}
		if err := s.Update(cmd.Context(), version); err != nil {
			return err
		}
		m, _, err := s.Status()
		if err != nil {
			return err
		}
		ui.Successf(cmd.OutOrStdout(), "updated to %s", m.Version)
		return nil
	},
}

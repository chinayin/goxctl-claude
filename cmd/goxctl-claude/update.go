package main

import "github.com/spf13/cobra"

var updateCmd = &cobra.Command{
	Use:   "update [version]",
	Short: "Pull the locked version (no arg) or upgrade to a given version",
	Long: `Sync managed files in one of two modes:

  no args     pull the version locked in .gox-claude.lock (restore on fresh clone / CI check, idempotent).
  <version>   upgrade to that tag (e.g. v1.1.0) and rewrite the manifest and lock.

After upgrading, review the .kiro/steering changes via git diff before committing; the version is pinned in .gox-claude.lock.`,
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
		return s.Update(cmd.Context(), version)
	},
}

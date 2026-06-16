package main

import (
	"github.com/chinayin/goxctl-claude/internal/ui"
	"github.com/spf13/cobra"
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Materialize managed files from the lock without changing it (CI / fresh clone)",
	Long: `install pulls exactly the commit pinned in .gox-claude.lock and writes the managed
files into the target dir, then verifies their digest. It never rewrites the
manifest or lock.

Use it when the managed files are not committed to git (gitignored): a fresh
clone or CI has only the manifest + lock and needs to restore the locked version
— the equivalent of 'npm ci'. To upgrade the pinned version, use 'update'.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		cmd.SilenceUsage = true
		s, err := newSyncer()
		if err != nil {
			return err
		}
		if err := s.Install(cmd.Context()); err != nil {
			return err
		}
		ui.Successf(cmd.OutOrStdout(), "installed from lock")
		return nil
	},
}

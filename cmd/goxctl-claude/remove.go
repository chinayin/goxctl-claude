package main

import (
	"github.com/chinayin/goxctl-claude/internal/ui"
	"github.com/spf13/cobra"
)

var removeCmd = &cobra.Command{
	Use:     "remove",
	Aliases: []string{"rm"},
	Short:   "Remove managed files and manifest/lock (leaves your own files untouched)",
	Args:    cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		cmd.SilenceUsage = true
		s, err := newSyncer()
		if err != nil {
			return err
		}
		if err := s.Remove(); err != nil {
			return err
		}
		ui.Successf(cmd.OutOrStdout(), "removed managed files and manifest/lock")
		return nil
	},
}

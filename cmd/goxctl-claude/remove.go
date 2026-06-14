package main

import "github.com/spf13/cobra"

var removeCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove managed files and manifest/lock (leaves your own files untouched)",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		cmd.SilenceUsage = true
		s, err := newSyncer()
		if err != nil {
			return err
		}
		return s.Remove()
	},
}

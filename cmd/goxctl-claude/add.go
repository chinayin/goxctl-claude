package main

import "github.com/spf13/cobra"

var (
	addVersion string
	addPaths   []string
	addTarget  string
)

var addCmd = &cobra.Command{
	Use:   "add <source>",
	Short: "Add a standards source and pull it (first-time init)",
	Long: `Add a standards source and pull it into the project (first-time init, writes .gox-claude.yaml).

<source> may be shortened to owner/repo (host defaults to github.com), or given in full as github.com/owner/repo.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		s, err := newSyncer()
		if err != nil {
			return err
		}
		return s.Add(cmd.Context(), args[0], addVersion, addPaths, addTarget)
	},
}

func init() {
	addCmd.Flags().StringVar(&addVersion, "version", "", "standards version tag, e.g. v1.0.0 (required)")
	addCmd.Flags().StringSliceVar(&addPaths, "paths", []string{"steering/"}, "directories/globs to sync")
	addCmd.Flags().StringVar(&addTarget, "target", "", "destination directory (default .kiro/steering)")
	_ = addCmd.MarkFlagRequired("version")
}

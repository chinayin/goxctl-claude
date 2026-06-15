package main

import (
	"fmt"

	"github.com/chinayin/goxctl-claude/internal/claude"
	"github.com/spf13/cobra"
)

var (
	addVersion string
	addPaths   []string
	addTarget  string
)

var addCmd = &cobra.Command{
	Use:   "add [source]",
	Short: "Add a standards source and pull it (first-time init)",
	Long: `Add a standards source and pull it into the project (first-time init, writes .gox-claude.yaml).

<source> defaults to ` + claude.DefaultSource + ` (the team standards repo); may be shortened to
owner/repo (host defaults to github.com), or given in full as github.com/owner/repo.

If --version is omitted, the latest release is resolved and pinned (still reproducible, not rolling latest).

Also generates a top-level CLAUDE.md entrypoint if the project has none (left untouched if it exists).`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		source := claude.DefaultSource
		if len(args) == 1 {
			source = args[0]
		}
		s, err := newSyncer()
		if err != nil {
			return err
		}
		created, err := s.Add(cmd.Context(), source, addVersion, addPaths, addTarget)
		if err != nil {
			return err
		}
		m, _, err := s.Status()
		if err != nil {
			return err
		}
		out := cmd.OutOrStdout()
		fmt.Fprintf(out, "added %s@%s -> %s\n", m.Source, m.Version, m.Target)
		if created {
			fmt.Fprintf(out, "generated %s\n", claude.ClaudeMdFile)
		} else {
			fmt.Fprintf(out, "%s left as-is (already present, or no template in this version)\n", claude.ClaudeMdFile)
		}
		return nil
	},
}

func init() {
	addCmd.Flags().StringVar(&addVersion, "version", "", "standards version tag, e.g. v0.1.0 (default: latest release)")
	addCmd.Flags().StringSliceVar(&addPaths, "paths", []string{"steering/"}, "directories/globs to sync")
	addCmd.Flags().StringVar(&addTarget, "target", "", "destination directory (default .kiro/steering)")
}

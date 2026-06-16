package main

import (
	"fmt"

	"github.com/chinayin/goxctl-claude/internal/ui"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "Show the current source, version and managed files",
	Args:    cobra.NoArgs,
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
		t := ui.Table(out)
		fmt.Fprintf(t, "source:\t%s\n", m.Source)
		fmt.Fprintf(t, "version:\t%s\n", m.Version)
		fmt.Fprintf(t, "target:\t%s\n", m.Target)
		if l != nil {
			fmt.Fprintf(t, "locked:\t%s @ %s\n", l.Version, l.Commit)
			fmt.Fprintf(t, "managed:\t%d files\n", len(l.Managed))
		} else {
			fmt.Fprintf(t, "locked:\t(not synced yet, run `update`)\n")
		}
		t.Flush()

		if l != nil {
			for _, f := range l.Managed {
				// 受管文件清单是补充信息，暗色显示（非表格行，不影响对齐）
				fmt.Fprintf(out, "  - %s\n", ui.Dim(f))
			}
		}
		return nil
	},
}

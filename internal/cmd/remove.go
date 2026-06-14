package cmd

import "github.com/spf13/cobra"

var removeCmd = &cobra.Command{
	Use:   "remove",
	Short: "移除受管文件与 manifest/lock（不碰项目自有文件）",
	Args:  cobra.NoArgs,
	RunE: func(_ *cobra.Command, _ []string) error {
		s, err := newSyncer()
		if err != nil {
			return err
		}
		return s.Remove()
	},
}

package cmd

import "github.com/spf13/cobra"

var updateCmd = &cobra.Command{
	Use:   "update [version]",
	Short: "拉到锁定版本（无参=恢复/校正）或升级到指定版本",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
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

package main

import "github.com/spf13/cobra"

var updateCmd = &cobra.Command{
	Use:   "update [version]",
	Short: "拉到锁定版本（无参=恢复/校正）或升级到指定版本",
	Long: `同步受管文件，有两种模式：

  无参数       拉到 .gox-claude.lock 锁定的版本（新 clone 恢复 / CI 校正，幂等）。
  指定 version 升级到该 tag（如 v1.1.0）并改写 manifest 与 lock。

升级后请 git diff 审阅 .kiro/steering 变化再提交，版本固定在 .gox-claude.lock。`,
	Args: cobra.MaximumNArgs(1),
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

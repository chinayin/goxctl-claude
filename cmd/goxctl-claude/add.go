package main

import "github.com/spf13/cobra"

var (
	addVersion string
	addPaths   []string
	addTarget  string
)

var addCmd = &cobra.Command{
	Use:   "add <source>",
	Short: "添加规范源并拉取（首次初始化）",
	Long: `添加规范源并拉取到本地（首次初始化，写 .goxctl-claude.yaml）。

source 可简写为 owner/repo（默认 github.com），也可写全 github.com/owner/repo。`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := newSyncer()
		if err != nil {
			return err
		}
		return s.Add(cmd.Context(), args[0], addVersion, addPaths, addTarget)
	},
}

func init() {
	addCmd.Flags().StringVar(&addVersion, "version", "", "规范版本 tag，如 v1.0.0（必填）")
	addCmd.Flags().StringSliceVar(&addPaths, "paths", []string{"steering/"}, "要同步的目录/glob")
	addCmd.Flags().StringVar(&addTarget, "target", "", "落地目录（缺省 .kiro/steering）")
	_ = addCmd.MarkFlagRequired("version")
}

package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/chinayin/goxctl-claude/internal/claude"
	"github.com/chinayin/goxctl-claude/internal/debug"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "claude",
	Short: "Sync team AI collaboration config (steering / CLAUDE.md)",
	Long: `claude syncs steering files from a standards repo by git tag into the local
project (default .kiro/steering), shared by Kiro and Claude Code; the version is
locked in .gox-claude.lock.

Typically used as a goxctl subcommand: goxctl claude <command>.`,
	// 不设 SilenceUsage：参数/flag 用法错误时显示 usage；业务错误在各 RunE 开头抑制。
	SilenceErrors: true, // 错误由 Execute 统一打印
}

// Execute 运行根命令，支持 Ctrl-C 中断。
func Execute() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	return rootCmd.ExecuteContext(ctx)
}

func init() {
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "enable verbose debug output (or set GOXCTL_DEBUG=1)")
	rootCmd.PersistentPreRun = func(cmd *cobra.Command, _ []string) {
		if v, _ := cmd.Flags().GetBool("verbose"); v {
			debug.Enable()
		}
	}
	rootCmd.AddCommand(addCmd, updateCmd, removeCmd, listCmd, checkCmd)
}

// newSyncer 基于当前工作目录与环境 token（GH_TOKEN / GITHUB_TOKEN）构建 Syncer。
func newSyncer() (*claude.Syncer, error) {
	dir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("claude: getwd: %w", err)
	}
	token := os.Getenv("GH_TOKEN")
	if token == "" {
		token = os.Getenv("GITHUB_TOKEN")
	}
	return claude.NewSyncer(dir, claude.NewFetcher(claude.WithToken(token))), nil
}

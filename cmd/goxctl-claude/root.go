package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/chinayin/goxctl-claude/internal/claude"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "claude",
	Short: "团队 AI 协作配置（steering / CLAUDE.md）版本化同步",
	Long: `claude 从规范源仓库按 git tag 同步 steering 文件到本地（默认 .kiro/steering），
供 Kiro 与 Claude Code 共用，版本锁定在 .goxctl-claude.lock。

通常作为 goxctl 的子命令使用：goxctl claude <command>。`,
	SilenceUsage:  true,
	SilenceErrors: true,
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

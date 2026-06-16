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

// version 在构建时通过 -ldflags "-X main.version=vX.Y.Z" 注入。
var version = "dev"

var rootCmd = &cobra.Command{
	Use:   "claude",
	Short: "Sync team AI collaboration config (steering / CLAUDE.md)",
	Long: `claude syncs steering files from a standards repo by git tag into the local
project (default .kiro/steering), shared by Kiro and Claude Code; the version is
locked in .gox-claude.lock.

Typically used as a goxctl subcommand: goxctl claude <command>.`,
	Example: `  # First-time init: pull the team standards and pin the latest release
  goxctl claude add

  # Upgrade to the latest standards release
  goxctl claude update

  # Verify local files still match the lock (CI)
  goxctl claude check`,
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

	// version 走 -V（大写）短旗，凑齐 -v/-V/-h 三件套。root 的 --version 为局部 flag，
	// 不会与 add 子命令的 --version（规范版本）冲突。
	rootCmd.Version = version
	rootCmd.Flags().BoolP("version", "V", false, "print version and exit")

	// 去噪：移除自动生成的 completion 子命令、隐藏 help 子命令。
	// cobra 默认模板用 (eq .Name "help") 硬编码强制列出 help，.Hidden 对它无效；
	// 故自定义一个真名非 "help" 的隐藏命令、用别名 "help" 接管。
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.SetHelpCommand(newHiddenHelpCmd())
	rootCmd.AddCommand(addCmd, updateCmd, removeCmd, listCmd, checkCmd)
}

// newHiddenHelpCmd 复刻 cobra 默认 help 命令的行为，但真名非 "help" 且 Hidden，
// 从而不出现在命令列表里（见 init 注释）。
func newHiddenHelpCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "help-topic [command]",
		Aliases: []string{"help"},
		Short:   "Help about any command",
		Hidden:  true,
		Run: func(c *cobra.Command, args []string) {
			target, _, err := c.Root().Find(args)
			if target == nil || err != nil {
				_ = c.Root().Usage()
				return
			}
			target.InitDefaultHelpFlag()
			target.InitDefaultVersionFlag()
			_ = target.Help()
		},
	}
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

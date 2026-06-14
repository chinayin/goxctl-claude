package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "校验本地受管文件与 lock 一致（CI 用）",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		s, err := newSyncer()
		if err != nil {
			return err
		}
		if err := s.Check(); err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), "ok: managed files match lock")
		return nil
	},
}

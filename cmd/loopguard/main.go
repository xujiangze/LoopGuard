package main

import (
	"fmt"
	"os"

	"LoopGuard/internal/cli"

	"github.com/spf13/cobra"
)

func main() {
	root := &cobra.Command{
		Use:   "loopguard",
		Short: "LoopGuard - AI 危险操作人工审批卡点平台",
	}
	root.AddCommand(cli.ServeCmd(), cli.MigrateCmd(), cli.AdminCmd(), cli.APIKeyCmd())
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

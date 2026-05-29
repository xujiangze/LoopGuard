package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	root := &cobra.Command{
		Use:   "loopguard",
		Short: "LoopGuard - AI 危险操作人工审批卡点平台",
	}
	// 子命令在后续 Task 注册：root.AddCommand(...)
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

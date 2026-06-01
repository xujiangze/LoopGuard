package cli

import (
	"testing"

	"github.com/spf13/cobra"
)

// 测试 opsx apply 命令存在且可执行
func TestOpsxApplyCommandExists(t *testing.T) {
	cmd := OpsxCmd()

	if cmd == nil {
		t.Fatal("OpsxCmd() 应该返回非 nil 的命令")
	}

	if cmd.Use != "opsx" {
		t.Errorf("期望命令名为 'opsx'，得到 '%s'", cmd.Use)
	}

	// 检查 apply 子命令存在
	var applyCmd *cobra.Command
	for _, sub := range cmd.Commands() {
		if sub.Use == "apply" {
			applyCmd = sub
			break
		}
	}

	if applyCmd == nil {
		t.Fatal("opsx 命令应该有 'apply' 子命令")
	}

	// 检查命令需要提案名参数
	if applyCmd.Args == nil {
		t.Error("apply 命令应该需要参数（提案名）")
	}
}
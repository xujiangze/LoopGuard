package service

import (
	"strconv"
	"strings"

	"LoopGuard/internal/executor"
)

const DryrunMarker = "DRYRUN-OK"

type DryrunResult struct {
	Passed bool
	Reason string
}

func ValidateDryrun(res *executor.ExecResult) DryrunResult {
	if res.TimedOut {
		return DryrunResult{false, "dry-run 执行超时"}
	}
	if res.ExitCode != 0 {
		return DryrunResult{false, "dry-run 退出码非 0（实际 " + strconv.Itoa(res.ExitCode) + "）"}
	}
	if !strings.Contains(res.Stdout, DryrunMarker) {
		return DryrunResult{false, "dry-run 输出缺少 " + DryrunMarker + " 标记，疑似未正确实现 --only-print"}
	}
	return DryrunResult{true, ""}
}

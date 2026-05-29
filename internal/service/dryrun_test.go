package service

import (
	"testing"

	"LoopGuard/internal/executor"

	"github.com/stretchr/testify/assert"
)

func TestValidateDryrun(t *testing.T) {
	ok := ValidateDryrun(&executor.ExecResult{ExitCode: 0, Stdout: "DRYRUN-OK\nwill delete x"})
	assert.True(t, ok.Passed)

	r1 := ValidateDryrun(&executor.ExecResult{ExitCode: 1, Stdout: "DRYRUN-OK"})
	assert.False(t, r1.Passed)
	assert.Contains(t, r1.Reason, "退出码")

	r2 := ValidateDryrun(&executor.ExecResult{ExitCode: 0, Stdout: "did something"})
	assert.False(t, r2.Passed)
	assert.Contains(t, r2.Reason, "DRYRUN-OK")

	r3 := ValidateDryrun(&executor.ExecResult{TimedOut: true})
	assert.False(t, r3.Passed)
}

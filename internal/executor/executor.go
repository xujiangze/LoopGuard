package executor

import "context"

type ExecRequest struct {
	BinaryPath string
	Args       []string
	DryRun     bool
	TimeoutSec int
	WorkDir    string
	Env        []string
}

type ExecResult struct {
	Command    string
	ExitCode   int
	Stdout     string
	Stderr     string
	DurationMs int64
	TimedOut   bool
}

type Executor interface {
	Run(ctx context.Context, req ExecRequest) (*ExecResult, error)
}

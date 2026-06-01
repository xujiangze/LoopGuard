package executor

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

type ProcessExecutor struct{}

func NewProcessExecutor() *ProcessExecutor { return &ProcessExecutor{} }

func (p *ProcessExecutor) Run(ctx context.Context, req ExecRequest) (*ExecResult, error) {
	args := append([]string{}, req.Args...)
	if req.DryRun {
		args = append(args, "--only-print")
	}

	timeout := time.Duration(req.TimeoutSec) * time.Second
	if timeout <= 0 {
		timeout = 300 * time.Second
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var cmd *exec.Cmd
	if req.Interpreter != "" {
		cmd = exec.CommandContext(runCtx, req.Interpreter, append([]string{req.BinaryPath}, args...)...)
	} else {
		cmd = exec.CommandContext(runCtx, req.BinaryPath, args...)
	}
	cmd.Dir = req.WorkDir
	cmd.Env = req.Env
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		if cmd.Process != nil {
			return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		}
		return nil
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	start := time.Now()
	err := cmd.Run()
	dur := time.Since(start).Milliseconds()

	res := &ExecResult{
		Command:    buildCommandString(req, args),
		Stdout:     stdout.String(),
		Stderr:     stderr.String(),
		DurationMs: dur,
	}

	if runCtx.Err() == context.DeadlineExceeded {
		res.TimedOut = true
		res.ExitCode = -1
		return res, nil
	}
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			res.ExitCode = ee.ExitCode()
			return res, nil
		}
		return res, err
	}
	res.ExitCode = 0
	return res, nil
}

func buildCommandString(req ExecRequest, args []string) string {
	if req.Interpreter != "" {
		return req.Interpreter + " " + req.BinaryPath + " " + strings.Join(args, " ")
	}
	return req.BinaryPath + " " + strings.Join(args, " ")
}

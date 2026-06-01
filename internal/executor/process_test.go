package executor

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProcessExecutorEcho(t *testing.T) {
	wd, _ := os.MkdirTemp("", "lg-test-*")
	defer os.RemoveAll(wd)
	e := NewProcessExecutor()
	res, err := e.Run(context.Background(), ExecRequest{
		BinaryPath: "/bin/echo",
		Args:       []string{"hello"},
		TimeoutSec: 5,
		WorkDir:    wd,
		Env:        []string{"PATH=/usr/bin:/bin"},
	})
	require.NoError(t, err)
	assert.Equal(t, 0, res.ExitCode)
	assert.Contains(t, res.Stdout, "hello")
	assert.False(t, res.TimedOut)
}

func TestProcessExecutorNonZeroExit(t *testing.T) {
	wd, _ := os.MkdirTemp("", "lg-test-*")
	defer os.RemoveAll(wd)
	e := NewProcessExecutor()
	res, err := e.Run(context.Background(), ExecRequest{
		BinaryPath: "/bin/sh",
		Args:       []string{"-c", "exit 3"},
		TimeoutSec: 5,
		WorkDir:    wd,
	})
	require.NoError(t, err)
	assert.Equal(t, 3, res.ExitCode)
}

func TestProcessExecutorTimeout(t *testing.T) {
	wd, _ := os.MkdirTemp("", "lg-test-*")
	defer os.RemoveAll(wd)
	e := NewProcessExecutor()
	res, err := e.Run(context.Background(), ExecRequest{
		BinaryPath: "/bin/sh",
		Args:       []string{"-c", "sleep 10"},
		TimeoutSec: 1,
		WorkDir:    wd,
	})
	require.NoError(t, err)
	assert.True(t, res.TimedOut)
}

func TestProcessExecutorWithInterpreter(t *testing.T) {
	wd, _ := os.MkdirTemp("", "lg-test-*")
	defer os.RemoveAll(wd)
	script := wd + "/test.py"
	err := os.WriteFile(script, []byte("import sys\nprint('hello from python')\n"), 0o644)
	require.NoError(t, err)

	e := NewProcessExecutor()
	res, err := e.Run(context.Background(), ExecRequest{
		Interpreter: "python3",
		BinaryPath:  script,
		Args:        []string{},
		TimeoutSec:  5,
		WorkDir:     wd,
	})
	require.NoError(t, err)
	assert.Equal(t, 0, res.ExitCode)
	assert.Contains(t, res.Stdout, "hello from python")
	assert.Contains(t, res.Command, "python3 "+script)
}

func TestProcessExecutorInterpreterDryRun(t *testing.T) {
	wd, _ := os.MkdirTemp("", "lg-test-*")
	defer os.RemoveAll(wd)
	script := wd + "/test.py"
	err := os.WriteFile(script, []byte("import sys\nprint('DRYRUN-OK')\nprint(sys.argv)\n"), 0o644)
	require.NoError(t, err)

	e := NewProcessExecutor()
	res, err := e.Run(context.Background(), ExecRequest{
		Interpreter: "python3",
		BinaryPath:  script,
		Args:        []string{"--env", "prod"},
		DryRun:      true,
		TimeoutSec:  5,
		WorkDir:     wd,
	})
	require.NoError(t, err)
	assert.Equal(t, 0, res.ExitCode)
	assert.Contains(t, res.Command, "python3 "+script+" --env prod --only-print")
}

func TestBuildCommandString(t *testing.T) {
	tests := []struct {
		name        string
		req         ExecRequest
		args        []string
		wantContain string
	}{
		{
			name:        "no interpreter",
			req:         ExecRequest{BinaryPath: "/usr/bin/tool"},
			args:        []string{"--env", "prod"},
			wantContain: "/usr/bin/tool --env prod",
		},
		{
			name:        "with interpreter",
			req:         ExecRequest{BinaryPath: "/app/deploy.py", Interpreter: "python3"},
			args:        []string{"--env", "prod"},
			wantContain: "python3 /app/deploy.py --env prod",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildCommandString(tt.req, tt.args)
			assert.Equal(t, tt.wantContain, got)
		})
	}
}

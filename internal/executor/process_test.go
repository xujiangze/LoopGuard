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

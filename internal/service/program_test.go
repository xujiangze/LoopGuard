package service

import (
	"context"
	"testing"

	"LoopGuard/internal/executor"
	"LoopGuard/internal/model"
	"LoopGuard/internal/store"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type fakeExecutor struct {
	result *executor.ExecResult
	err    error
}

func (f *fakeExecutor) Run(_ context.Context, _ executor.ExecRequest) (*executor.ExecResult, error) {
	return f.result, f.err
}

func newTestProgramService(t *testing.T, exec executor.Executor) (*ProgramService, *store.Store) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	s := store.New(db)
	require.NoError(t, s.AutoMigrate())
	return NewProgramService(s, exec, "/tmp/test-workspace"), s
}

func TestRegisterProgramValidation(t *testing.T) {
	fe := &fakeExecutor{result: &executor.ExecResult{ExitCode: 0, Stdout: "usage: deploy [--only-print]"}}
	svc, s := newTestProgramService(t, fe)

	u := &model.User{Username: "appr", PasswordHash: "h", Role: model.RoleUser}
	require.NoError(t, s.CreateUser(u))

	_, err := svc.Register(context.Background(), RegisterInput{
		Project: "demo", Name: "deploy", EntryFile: "deploy.sh",
		ApproverID: u.ID, TimeoutSec: 60,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "interpreter")
}

func TestRegisterProgramRejectsBadProjectName(t *testing.T) {
	fe := &fakeExecutor{result: &executor.ExecResult{ExitCode: 0, Stdout: "usage"}}
	svc, _ := newTestProgramService(t, fe)

	_, err := svc.Register(context.Background(), RegisterInput{
		Project: "my/project", Name: "deploy", EntryFile: "deploy.sh",
		Interpreter: "bash",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "非法字符")
}

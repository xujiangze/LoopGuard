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
	return NewProgramService(s, exec), s
}

func TestRegisterProgramHelpProbe(t *testing.T) {
	fe := &fakeExecutor{result: &executor.ExecResult{ExitCode: 0, Stdout: "usage: deploy [--only-print]"}}
	svc, s := newTestProgramService(t, fe)

	u := &model.User{Username: "appr", PasswordHash: "h", Role: model.RoleUser}
	require.NoError(t, s.CreateUser(u))

	p, err := svc.Register(context.Background(), RegisterInput{
		Project: "demo", Name: "deploy", BinaryPath: "/bin/deploy",
		ApproverID: u.ID, TimeoutSec: 60, ParamsSchema: []byte(`{"env":"string"}`),
	})
	require.NoError(t, err)
	require.NotZero(t, p.ID)
	require.NotEmpty(t, p.HelpText)
	require.True(t, p.SupportsDryrun)
}

func TestRegisterProgramRejectsUnknownFlag(t *testing.T) {
	fe := &fakeExecutor{result: &executor.ExecResult{ExitCode: 2, Stderr: "unknown flag: --only-print"}}
	svc, s := newTestProgramService(t, fe)
	u := &model.User{Username: "appr2", PasswordHash: "h", Role: model.RoleUser}
	require.NoError(t, s.CreateUser(u))

	_, err := svc.Register(context.Background(), RegisterInput{
		Project: "demo", Name: "bad", BinaryPath: "/bin/bad", ApproverID: u.ID,
	})
	require.Error(t, err)
}

func TestRegisterProgramWithInterpreter(t *testing.T) {
	fe := &fakeExecutor{result: &executor.ExecResult{ExitCode: 0, Stdout: "usage: deploy [--only-print]"}}
	svc, s := newTestProgramService(t, fe)

	u := &model.User{Username: "appr3", PasswordHash: "h", Role: model.RoleUser}
	require.NoError(t, s.CreateUser(u))

	p, err := svc.Register(context.Background(), RegisterInput{
		Project: "demo", Name: "py-deploy", BinaryPath: "/app/deploy.py", Interpreter: "python3",
		ApproverID: u.ID, TimeoutSec: 60,
	})
	require.NoError(t, err)
	require.Equal(t, "python3", p.Interpreter)
	require.Equal(t, "/app/deploy.py", p.BinaryPath)
}

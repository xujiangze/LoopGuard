package service

import (
	"context"
	"testing"

	"LoopGuard/internal/executor"
	"LoopGuard/internal/model"
	"LoopGuard/internal/store"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newTicketService(t *testing.T, exec executor.Executor) (*TicketService, *store.Store) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=private"), &gorm.Config{})
	require.NoError(t, err)
	s := store.New(db)
	require.NoError(t, s.AutoMigrate())
	return NewTicketService(s, exec), s
}

func seedProgram(t *testing.T, s *store.Store, schema string) *model.Program {
	u := &model.User{Username: "appr", PasswordHash: "h", Role: model.RoleUser}
	require.NoError(t, s.CreateUser(u))
	p := &model.Program{Project: "demo", Name: "deploy", BinaryPath: "/bin/deploy",
		ApproverID: u.ID, TimeoutSec: 30, SupportsDryrun: true, Enabled: true,
		ParamsSchema: []byte(schema)}
	require.NoError(t, s.CreateProgram(p))
	return p
}

func TestSubmitTicketDryrunPass(t *testing.T) {
	fe := &fakeExecutor{result: &executor.ExecResult{ExitCode: 0, Stdout: "DRYRUN-OK\nwill deploy"}}
	svc, s := newTicketService(t, fe)
	seedProgram(t, s, `{"env":"string"}`)

	tk, err := svc.Submit(context.Background(), SubmitInput{
		Project: "demo", Name: "deploy", APIKeyID: 7,
		Args: map[string]any{"env": "prod"},
	})
	require.NoError(t, err)
	assert.Equal(t, model.StatusPendingApproval, tk.Status)
	assert.Contains(t, tk.DryrunOutput, "DRYRUN-OK")
}

func TestSubmitTicketDryrunFail(t *testing.T) {
	fe := &fakeExecutor{result: &executor.ExecResult{ExitCode: 0, Stdout: "no marker"}}
	svc, s := newTicketService(t, fe)
	seedProgram(t, s, `{"env":"string"}`)

	tk, err := svc.Submit(context.Background(), SubmitInput{
		Project: "demo", Name: "deploy", APIKeyID: 7, Args: map[string]any{"env": "prod"},
	})
	require.NoError(t, err)
	assert.Equal(t, model.StatusDryrunFailed, tk.Status)
}

func TestSubmitRejectsUnknownArg(t *testing.T) {
	fe := &fakeExecutor{result: &executor.ExecResult{ExitCode: 0, Stdout: "DRYRUN-OK"}}
	svc, s := newTicketService(t, fe)
	seedProgram(t, s, `{"env":"string"}`)

	_, err := svc.Submit(context.Background(), SubmitInput{
		Project: "demo", Name: "deploy", APIKeyID: 7,
		Args: map[string]any{"env": "prod", "force": "true"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "force")
}

func TestSubmitRejectsOnlyPrintInArgs(t *testing.T) {
	fe := &fakeExecutor{result: &executor.ExecResult{ExitCode: 0, Stdout: "DRYRUN-OK"}}
	svc, s := newTicketService(t, fe)
	seedProgram(t, s, `{"only-print":"bool"}`)

	_, err := svc.Submit(context.Background(), SubmitInput{
		Project: "demo", Name: "deploy", APIKeyID: 7,
		Args: map[string]any{"only-print": "false"},
	})
	require.Error(t, err)
}

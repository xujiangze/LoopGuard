package service

import (
	"context"
	"errors"
	"testing"

	"LoopGuard/internal/executor"
	"LoopGuard/internal/model"
	"LoopGuard/internal/store"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newTicketService(t *testing.T, exec executor.Executor) (*TicketService, *store.Store) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=private"), &gorm.Config{})
	require.NoError(t, err)
	s := store.New(db)
	require.NoError(t, s.AutoMigrate())
	return NewTicketService(s, exec, "/tmp/test-workspace"), s
}

func seedProgram(t *testing.T, s *store.Store) *model.Program {
	u := &model.User{Username: "appr", PasswordHash: "h", Role: model.RoleUser}
	require.NoError(t, s.CreateUser(u))
	p := &model.Program{Project: "demo", Name: "deploy", EntryFile: "deploy.sh",
		Interpreter: "bash", ApproverID: u.ID, TimeoutSec: 30, SupportsDryrun: true, Enabled: true}
	require.NoError(t, s.CreateProgram(p))
	return p
}

func TestSubmitTicketDryrunPass(t *testing.T) {
	fe := &fakeExecutor{result: &executor.ExecResult{
		Command: "bash deploy.sh -e prod --only-print",
		ExitCode: 0, Stdout: "DRYRUN-OK\nwill deploy", Stderr: "",
	}}
	svc, s := newTicketService(t, fe)
	seedProgram(t, s)

	tk, err := svc.Submit(context.Background(), SubmitInput{
		Project: "demo", Name: "deploy", APIKeyID: 7,
		Args: []string{"-e", "prod"},
	})
	require.NoError(t, err)
	assert.Equal(t, model.StatusPendingApproval, tk.Status)
	assert.Contains(t, tk.DryrunOutput, "# 命令")
	assert.Contains(t, tk.DryrunOutput, "DRYRUN-OK")
	assert.Contains(t, tk.DryrunOutput, "校验: 通过")
}

func TestSubmitTicketDryrunFail(t *testing.T) {
	fe := &fakeExecutor{result: &executor.ExecResult{
		Command: "bash deploy.sh -e prod --only-print",
		ExitCode: 0, Stdout: "no marker", Stderr: "",
	}}
	svc, s := newTicketService(t, fe)
	seedProgram(t, s)

	tk, err := svc.Submit(context.Background(), SubmitInput{
		Project: "demo", Name: "deploy", APIKeyID: 7,
		Args: []string{"-e", "prod"},
	})
	require.NoError(t, err)
	assert.Equal(t, model.StatusDryrunFailed, tk.Status)
	assert.Contains(t, tk.DryrunOutput, "校验: 失败")
}

func TestSubmitRejectsDangerousArg(t *testing.T) {
	fe := &fakeExecutor{result: &executor.ExecResult{ExitCode: 0, Stdout: "DRYRUN-OK"}}
	svc, s := newTicketService(t, fe)
	seedProgram(t, s)

	_, err := svc.Submit(context.Background(), SubmitInput{
		Project: "demo", Name: "deploy", APIKeyID: 7,
		Args: []string{"-e", "prod; rm -rf /"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "危险字符")
}

func TestSubmitRejectsOnlyPrintInArgs(t *testing.T) {
	fe := &fakeExecutor{result: &executor.ExecResult{ExitCode: 0, Stdout: "DRYRUN-OK"}}
	svc, s := newTicketService(t, fe)
	seedProgram(t, s)

	_, err := svc.Submit(context.Background(), SubmitInput{
		Project: "demo", Name: "deploy", APIKeyID: 7,
		Args: []string{"--only-print"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "保留")
}

func TestSubmitWithInterpreter(t *testing.T) {
	fe := &fakeExecutor{result: &executor.ExecResult{ExitCode: 0, Stdout: "DRYRUN-OK\npython output"}}
	svc, s := newTicketService(t, fe)
	prog := seedProgram(t, s)
	prog.Interpreter = "python3"
	require.NoError(t, s.UpdateProgram(prog))

	tk, err := svc.Submit(context.Background(), SubmitInput{
		Project: "demo", Name: "deploy", APIKeyID: 7,
		Args: []string{"-e", "prod"},
	})
	require.NoError(t, err)
	assert.Equal(t, model.StatusPendingApproval, tk.Status)
}

func TestFormatExecReport(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		stdout   string
		stderr   string
		exitCode int
		result   string
		want     []string
	}{
		{
			name: "dryrun pass", command: "bash deploy.sh -e prod --only-print",
			stdout: "DRYRUN-OK", stderr: "", exitCode: 0, result: "校验: 通过",
			want: []string{"# 命令\nbash deploy.sh", "# stdout\nDRYRUN-OK", "# stderr\n(无)", "校验: 通过"},
		},
		{
			name: "exec fail", command: "bash deploy.sh -e prod",
			stdout: "", stderr: "error: connection refused", exitCode: 1, result: "耗时: 500ms",
			want: []string{"# stderr\nerror: connection refused", "退出码: 1"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatExecReport(tt.command, tt.stdout, tt.stderr, tt.exitCode, tt.result)
			for _, w := range tt.want {
				assert.Contains(t, got, w)
			}
		})
	}
}

func TestSubmitTicketDryrunRunError(t *testing.T) {
	fe := &fakeExecutor{result: nil, err: errors.New("binary not found")}
	svc, s := newTicketService(t, fe)
	seedProgram(t, s)

	tk, err := svc.Submit(context.Background(), SubmitInput{
		Project: "demo", Name: "deploy", APIKeyID: 7,
		Args: []string{"-e", "prod"},
	})
	require.NoError(t, err)
	assert.Equal(t, model.StatusDryrunFailed, tk.Status)
	assert.Contains(t, tk.DryrunOutput, "binary not found")
}

func TestApproveFillsExecOutput(t *testing.T) {
	fe := &fakeExecutor{result: &executor.ExecResult{
		Command: "bash deploy.sh -e prod", ExitCode: 0,
		Stdout: "deployment configured", Stderr: "", DurationMs: 1523,
	}}
	svc, s := newTicketService(t, fe)
	p := seedProgram(t, s)

	tk := &model.Ticket{
		ProgramID: p.ID, Args: datatypes.JSON(`["-e","prod"]`),
		Status: model.StatusPendingApproval, SubmittedBy: 7,
		ApproverID: p.ApproverID,
		DryrunOutput: "# 命令\nbash deploy.sh --only-print\n\n# 结果\n退出码: 0 | 校验: 通过",
	}
	require.NoError(t, s.CreateTicket(tk))

	result, err := svc.Approve(context.Background(), tk.ID, p.ApproverID)
	require.NoError(t, err)
	assert.Equal(t, model.StatusDone, result.Status)
	assert.Contains(t, result.ExecOutput, "deployment configured")
	assert.Contains(t, result.ExecOutput, "耗时: 1523ms")
}

func TestApproveExecFailedFillsExecOutput(t *testing.T) {
	fe := &fakeExecutor{result: &executor.ExecResult{
		Command: "bash deploy.sh -e prod", ExitCode: 1,
		Stdout: "", Stderr: "error: connection refused", DurationMs: 500,
	}}
	svc, s := newTicketService(t, fe)
	p := seedProgram(t, s)

	tk := &model.Ticket{
		ProgramID: p.ID, Args: datatypes.JSON(`["-e","prod"]`),
		Status: model.StatusPendingApproval, SubmittedBy: 7,
		ApproverID: p.ApproverID,
		DryrunOutput: "# 命令\nbash deploy.sh --only-print",
	}
	require.NoError(t, s.CreateTicket(tk))

	result, err := svc.Approve(context.Background(), tk.ID, p.ApproverID)
	require.NoError(t, err)
	assert.Equal(t, model.StatusExecFailed, result.Status)
	assert.Contains(t, result.ExecOutput, "error: connection refused")
	assert.Contains(t, result.ExecOutput, "退出码: 1")
}

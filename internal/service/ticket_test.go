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

func TestSubmitWithInterpreter(t *testing.T) {
	fe := &fakeExecutor{result: &executor.ExecResult{ExitCode: 0, Stdout: "DRYRUN-OK\npython output"}}
	svc, s := newTicketService(t, fe)
	prog := seedProgram(t, s, `{"env":"string"}`)
	prog.Interpreter = "python3"
	require.NoError(t, s.UpdateProgram(prog))

	tk, err := svc.Submit(context.Background(), SubmitInput{
		Project: "demo", Name: "deploy", APIKeyID: 7,
		Args: map[string]any{"env": "prod"},
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
			name:     "dryrun pass with output",
			command:  "python3 /bin/deploy --env prod --only-print",
			stdout:   "将执行: kubectl apply\nDRYRUN-OK",
			stderr:   "",
			exitCode: 0,
			result:   "校验: 通过",
			want: []string{
				"# 命令\npython3 /bin/deploy --env prod --only-print",
				"# stdout\n将执行: kubectl apply\nDRYRUN-OK",
				"# stderr\n(无)",
				"# 结果\n退出码: 0 | 校验: 通过",
			},
		},
		{
			name:     "exec fail with stderr",
			command:  "/bin/deploy --env prod",
			stdout:   "",
			stderr:   "error: connection refused",
			exitCode: 1,
			result:   "校验: 失败 - dry-run 退出码非 0（实际 1）",
			want: []string{
				"# 命令\n/bin/deploy --env prod",
				"# stdout\n",
				"# stderr\nerror: connection refused",
				"# 结果\n退出码: 1 | 校验: 失败 - dry-run 退出码非 0（实际 1）",
			},
		},
		{
			name:     "real exec with duration",
			command:  "python3 /bin/deploy --env prod",
			stdout:   "deployment configured",
			stderr:   "",
			exitCode: 0,
			result:   "耗时: 1523ms",
			want: []string{
				"# 命令\npython3 /bin/deploy --env prod",
				"# stdout\ndeployment configured",
				"# stderr\n(无)",
				"# 结果\n退出码: 0 | 耗时: 1523ms",
			},
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

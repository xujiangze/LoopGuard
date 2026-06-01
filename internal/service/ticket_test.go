package service

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"testing"
	"time"

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

type capturingExecutor struct {
	result  *executor.ExecResult
	err     error
	lastReq executor.ExecRequest
}

func (c *capturingExecutor) Run(_ context.Context, req executor.ExecRequest) (*executor.ExecResult, error) {
	c.lastReq = req
	return c.result, c.err
}

func TestSubmitBinaryPathNoDuplicationWithRelativeWorkspace(t *testing.T) {
	ce := &capturingExecutor{result: &executor.ExecResult{
		ExitCode: 0, Stdout: "DRYRUN-OK\nwill unban",
	}}
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=private"), &gorm.Config{})
	require.NoError(t, err)
	s := store.New(db)
	require.NoError(t, s.AutoMigrate())
	// 使用相对路径 workspaceDir，暴露路径重复 bug
	svc := NewTicketService(s, ce, "./workspace")

	u := &model.User{Username: "appr", PasswordHash: "h", Role: model.RoleUser}
	require.NoError(t, s.CreateUser(u))
	p := &model.Program{
		Project: "tsunami_ipban", Name: "entry_ipban", EntryFile: "entry_ipban.py",
		Interpreter: "python3", ApproverID: u.ID, TimeoutSec: 30, SupportsDryrun: true, Enabled: true,
	}
	require.NoError(t, s.CreateProgram(p))

	_, err = svc.Submit(context.Background(), SubmitInput{
		Project: "tsunami_ipban", Name: "entry_ipban", APIKeyID: 1,
		Args: []string{"-m", "group_unban"},
	})
	require.NoError(t, err)

	// BinaryPath 应该只是文件名，因为 WorkDir 已经指向程序目录
	assert.Equal(t, "entry_ipban.py", ce.lastReq.BinaryPath)
	assert.Equal(t, filepath.Join("./workspace", "tsunami_ipban", "entry_ipban"), ce.lastReq.WorkDir)
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

func TestSubmitWithWebhookFailure(t *testing.T) {
	fe := &fakeExecutor{result: &executor.ExecResult{
		Command: "bash deploy.sh -e prod --only-print",
		ExitCode: 0, Stdout: "DRYRUN-OK\nwill deploy", Stderr: "",
	}}
	svc, s := newTicketService(t, fe)

	// 创建 WebhookService 并注入
	ws := NewWebhookService(s)
	svc.SetWebhook(ws, "http://localhost:8080")

	// 创建一个 WebhookConfig，URL 指向不存在的服务器
	wh := &model.WebhookConfig{
		ProgramID:  0, // 不限定程序，匹配所有
		Name:       "test-webhook",
		URL:        "http://127.0.0.1:1/nonexistent", // 确保连接失败
		Enabled:    true,
		EventTypes: "pending_approval,dryrun_failed",
	}
	require.NoError(t, s.CreateWebhook(wh))

	// 创建 Program 种子数据
	seedProgram(t, s)

	// 提交工单
	tk, err := svc.Submit(context.Background(), SubmitInput{
		Project: "demo", Name: "deploy", APIKeyID: 7,
		Args: []string{"-e", "prod"},
	})
	require.NoError(t, err)

	// 验证工单状态不受 webhook 失败影响
	assert.Equal(t, model.StatusPendingApproval, tk.Status)
}

func TestApproveTriggersWebhook(t *testing.T) {
	fe := &fakeExecutor{result: &executor.ExecResult{
		Command: "bash deploy.sh -e prod", ExitCode: 0,
		Stdout: "deployed", Stderr: "", DurationMs: 100,
	}}
	svc, s := newTicketService(t, fe)

	// 创建 mock webhook server
	var mu sync.Mutex
	var receivedBody []byte
	var requestReceived bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		mu.Lock()
		receivedBody = body
		requestReceived = true
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	// 创建 WebhookService 并注入
	ws := NewWebhookService(s)
	svc.SetWebhook(ws, "http://localhost:8080")

	// 创建 WebhookConfig，URL 指向 mock server
	wh := &model.WebhookConfig{
		ProgramID:  0,
		Name:       "test-webhook",
		URL:        server.URL,
		Enabled:    true,
		EventTypes: "done,exec_failed",
	}
	require.NoError(t, s.CreateWebhook(wh))

	// 创建 pending_approval 状态的工单
	p := seedProgram(t, s)
	tk := &model.Ticket{
		ProgramID: p.ID, Args: datatypes.JSON(`["-e","prod"]`),
		Status: model.StatusPendingApproval, SubmittedBy: 7,
		ApproverID: p.ApproverID,
		DryrunOutput: "# 命令\nbash deploy.sh --only-print\n\n# 结果\n退出码: 0 | 校验: 通过",
	}
	require.NoError(t, s.CreateTicket(tk))

	// 调用 Approve
	result, err := svc.Approve(context.Background(), tk.ID, p.ApproverID)
	require.NoError(t, err)
	assert.Equal(t, model.StatusDone, result.Status)

	// 等待异步 webhook 触发
	require.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return requestReceived
	}, 2*time.Second, 50*time.Millisecond, "mock server 应该收到 webhook 请求")

	// 验证收到的请求内容
	mu.Lock()
	defer mu.Unlock()
	var msg map[string]interface{}
	require.NoError(t, json.Unmarshal(receivedBody, &msg))
	assert.Equal(t, "text", msg["msgtype"])

	// 验证投递记录被创建
	deliveries, err := s.GetWebhookDeliveries(wh.ID)
	require.NoError(t, err)
	require.Len(t, deliveries, 1)
	assert.Equal(t, 200, deliveries[0].StatusCode)
	assert.Equal(t, "ok", deliveries[0].Response)
	assert.Equal(t, string(model.StatusDone), deliveries[0].EventType)
}

func TestRejectTriggersWebhook(t *testing.T) {
	fe := &fakeExecutor{result: &executor.ExecResult{ExitCode: 0, Stdout: "DRYRUN-OK"}}
	svc, s := newTicketService(t, fe)

	// 创建 mock webhook server
	var mu sync.Mutex
	var receivedBody []byte
	var requestReceived bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		mu.Lock()
		receivedBody = body
		requestReceived = true
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	// 创建 WebhookService 并注入
	ws := NewWebhookService(s)
	svc.SetWebhook(ws, "http://localhost:8080")

	// 创建 WebhookConfig
	wh := &model.WebhookConfig{
		ProgramID:  0,
		Name:       "test-webhook",
		URL:        server.URL,
		Enabled:    true,
		EventTypes: "rejected",
	}
	require.NoError(t, s.CreateWebhook(wh))

	// 创建 pending_approval 状态的工单
	p := seedProgram(t, s)
	tk := &model.Ticket{
		ProgramID: p.ID, Args: datatypes.JSON(`["-e","prod"]`),
		Status: model.StatusPendingApproval, SubmittedBy: 7,
		ApproverID: p.ApproverID,
		DryrunOutput: "# 命令\nbash deploy.sh --only-print\n\n# 结果\n退出码: 0 | 校验: 通过",
	}
	require.NoError(t, s.CreateTicket(tk))

	// 调用 Reject
	result, err := svc.Reject(tk.ID, p.ApproverID, "参数有误")
	require.NoError(t, err)
	assert.Equal(t, model.StatusRejected, result.Status)

	// 等待异步 webhook 触发
	require.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return requestReceived
	}, 2*time.Second, 50*time.Millisecond, "mock server 应该收到 webhook 请求")

	// 验证收到的请求内容
	mu.Lock()
	defer mu.Unlock()
	var msg map[string]interface{}
	require.NoError(t, json.Unmarshal(receivedBody, &msg))
	assert.Equal(t, "text", msg["msgtype"])

	// 验证投递记录被创建
	deliveries, err := s.GetWebhookDeliveries(wh.ID)
	require.NoError(t, err)
	require.Len(t, deliveries, 1)
	assert.Equal(t, 200, deliveries[0].StatusCode)
	assert.Equal(t, string(model.StatusRejected), deliveries[0].EventType)
}

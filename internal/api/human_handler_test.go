package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"LoopGuard/internal/auth"
	"LoopGuard/internal/config"
	"LoopGuard/internal/executor"
	"LoopGuard/internal/model"
	"LoopGuard/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeExecutor struct {
	result *executor.ExecResult
	err    error
}

func (f *fakeExecutor) Run(_ context.Context, _ executor.ExecRequest) (*executor.ExecResult, error) {
	return f.result, f.err
}

func mustReq(method, path string, body []byte) *http.Request {
	if body == nil {
		return httptest.NewRequest(method, path, nil)
	}
	r := httptest.NewRequest(method, path, bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	return r
}

func TestLoginAndApprove(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s := testStore(t)
	pw, _ := auth.HashPassword("pw123")
	appr := &model.User{Username: "appr", PasswordHash: pw, Role: model.RoleUser}
	require.NoError(t, s.CreateUser(appr))
	require.NoError(t, s.CreateProgram(&model.Program{Project: "demo", Name: "deploy",
		EntryFile: "deploy.sh", Interpreter: "bash", ApproverID: appr.ID, TimeoutSec: 10, SupportsDryrun: true,
		Enabled: true}))

	ex := &fakeExecutor{result: &executor.ExecResult{
		Command: "bash /tmp/test-ws/demo/deploy.sh msg DRYRUN-OK --only-print",
		ExitCode: 0, Stdout: "DRYRUN-OK", Stderr: "",
	}}
	ticketSvc := service.NewTicketService(s, ex, "/tmp/test-ws")
	tk, err := ticketSvc.Submit(context.Background(), service.SubmitInput{
		Project: "demo", Name: "deploy", APIKeyID: 1, Args: []string{"msg", "DRYRUN-OK"}})
	require.NoError(t, err)
	require.Equal(t, model.StatusPendingApproval, tk.Status)

	// Mock 实际执行
	ex.result = &executor.ExecResult{
		Command: "bash /tmp/test-ws/demo/deploy.sh msg DRYRUN-OK",
		ExitCode: 0, Stdout: "deployed", Stderr: "",
	}

	cfg := config.Config{JWTSecret: "s"}
	h := NewHumanHandler(s, ticketSvc, cfg)
	r := gin.New()
	r.POST("/login", h.Login)
	authGrp := r.Group("/", JWTAuth(cfg.JWTSecret))
	authGrp.POST("/tickets/:id/approve", h.Approve)

	body, _ := json.Marshal(map[string]string{"username": "appr", "password": "pw123"})
	lw := httptest.NewRecorder()
	r.ServeHTTP(lw, mustReq("POST", "/login", body))
	require.Equal(t, http.StatusOK, lw.Code)
	var lr map[string]any
	require.NoError(t, json.Unmarshal(lw.Body.Bytes(), &lr))
	token, _ := lr["token"].(string)
	require.NotEmpty(t, token)

	aw := httptest.NewRecorder()
	areq := mustReq("POST", "/tickets/"+fmt.Sprint(tk.ID)+"/approve", nil)
	areq.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(aw, areq)
	assert.Equal(t, http.StatusOK, aw.Code)

	got, _ := s.GetTicket(tk.ID)
	assert.Equal(t, model.StatusDone, got.Status)
}

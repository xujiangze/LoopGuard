package api

import (
	"bytes"
	"encoding/json"
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

// setupHumanSubmit 创建测试路由，模拟 JWT 认证后的手工提交场景
func setupHumanSubmit(t *testing.T) (*gin.Engine, *fakeExecutor) {
	gin.SetMode(gin.TestMode)
	s := testStore(t)

	// 创建审批人
	pw, _ := auth.HashPassword("pw123")
	appr := &model.User{Username: "appr", PasswordHash: pw, Role: model.RoleUser}
	require.NoError(t, s.CreateUser(appr))

	// 创建已启用 API Key
	require.NoError(t, s.CreateAPIKey(&model.APIKey{Name: "test-key", KeyHash: "hash1", Enabled: true}))

	// 注册程序
	require.NoError(t, s.CreateProgram(&model.Program{
		Project: "demo", Name: "deploy",
		EntryFile: "deploy.sh", Interpreter: "bash",
		ApproverID: appr.ID, TimeoutSec: 10,
		SupportsDryrun: true, Enabled: true,
	}))

	fe := &fakeExecutor{result: &executor.ExecResult{
		Command:  "bash deploy.sh --only-print",
		ExitCode: 0, Stdout: "DRYRUN-OK", Stderr: "",
	}}
	ticketSvc := service.NewTicketService(s, fe, "/tmp/test-ws")
	cfg := config.Config{JWTSecret: "s", BaseURL: "http://test"}
	h := NewHumanHandler(s, ticketSvc, cfg)

	r := gin.New()
	r.POST("/tickets/submit", JWTAuth(cfg.JWTSecret), h.Submit)
	return r, fe
}

func TestHumanSubmit_Success(t *testing.T) {
	r, _ := setupHumanSubmit(t)
	tok, _ := auth.SignJWT("s", 1, "user")

	body, _ := json.Marshal(map[string]any{
		"api_key_id": 1, "project": "demo", "name": "deploy",
		"args": []string{"--env", "prod"},
	})
	req := httptest.NewRequest("POST", "/tickets/submit", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotZero(t, resp["ticket_id"])
	assert.Equal(t, "pending_approval", resp["status"])
	assert.NotEmpty(t, resp["dryrun_output"])
}

func TestHumanSubmit_APIKeyNotExist(t *testing.T) {
	r, _ := setupHumanSubmit(t)
	tok, _ := auth.SignJWT("s", 1, "user")

	body, _ := json.Marshal(map[string]any{
		"api_key_id": 999, "project": "demo", "name": "deploy",
		"args": []string{},
	})
	req := httptest.NewRequest("POST", "/tickets/submit", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "API Key")
}

func TestHumanSubmit_APIKeyDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s := testStore(t)

	pw, _ := auth.HashPassword("pw123")
	appr := &model.User{Username: "appr", PasswordHash: pw, Role: model.RoleUser}
	require.NoError(t, s.CreateUser(appr))

	// 创建 API Key 后禁用
	key := &model.APIKey{Name: "disabled-key", KeyHash: "hash2", Enabled: true}
	require.NoError(t, s.CreateAPIKey(key))
	key.Enabled = false
	require.NoError(t, s.UpdateAPIKey(key))
	require.NoError(t, s.CreateProgram(&model.Program{
		Project: "demo", Name: "deploy",
		EntryFile: "deploy.sh", Interpreter: "bash",
		ApproverID: appr.ID, TimeoutSec: 10,
		SupportsDryrun: true, Enabled: true,
	}))

	fe := &fakeExecutor{result: &executor.ExecResult{
		Command: "bash deploy.sh --only-print", ExitCode: 0, Stdout: "DRYRUN-OK",
	}}
	ticketSvc := service.NewTicketService(s, fe, "/tmp/test-ws")
	cfg := config.Config{JWTSecret: "s", BaseURL: "http://test"}
	h := NewHumanHandler(s, ticketSvc, cfg)

	r := gin.New()
	r.POST("/tickets/submit", JWTAuth(cfg.JWTSecret), h.Submit)

	tok, _ := auth.SignJWT("s", 1, "user")
	body, _ := json.Marshal(map[string]any{
		"api_key_id": 1, "project": "demo", "name": "deploy",
		"args": []string{},
	})
	req := httptest.NewRequest("POST", "/tickets/submit", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "API Key")
}

func TestHumanSubmit_Unauthorized(t *testing.T) {
	r, _ := setupHumanSubmit(t)

	body, _ := json.Marshal(map[string]any{
		"api_key_id": 1, "project": "demo", "name": "deploy",
		"args": []string{},
	})
	req := httptest.NewRequest("POST", "/tickets/submit", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

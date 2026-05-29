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
		BinaryPath: "/bin/echo", ApproverID: appr.ID, TimeoutSec: 10, SupportsDryrun: true,
		Enabled: true, ParamsSchema: []byte(`{"msg":"string"}`)}))

	ex := executor.NewProcessExecutor()
	ticketSvc := service.NewTicketService(s, ex)
	tk, err := ticketSvc.Submit(context.Background(), service.SubmitInput{
		Project: "demo", Name: "deploy", APIKeyID: 1, Args: map[string]any{"msg": "DRYRUN-OK"}})
	require.NoError(t, err)
	require.Equal(t, model.StatusPendingApproval, tk.Status)

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

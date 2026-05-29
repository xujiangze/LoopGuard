package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"LoopGuard/internal/config"
	"LoopGuard/internal/executor"
	"LoopGuard/internal/model"
	"LoopGuard/internal/service"
	"LoopGuard/internal/store"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupAI(t *testing.T) (*gin.Engine, *store.Store) {
	gin.SetMode(gin.TestMode)
	s := testStore(t)
	u := &model.User{Username: "appr", PasswordHash: "h", Role: model.RoleUser}
	require.NoError(t, s.CreateUser(u))
	require.NoError(t, s.CreateProgram(&model.Program{Project: "demo", Name: "deploy",
		BinaryPath: "/bin/echo", ApproverID: u.ID, TimeoutSec: 10, SupportsDryrun: true,
		Enabled: true, ParamsSchema: []byte(`{"msg":"string"}`)}))

	ex := executor.NewProcessExecutor()
	ticketSvc := service.NewTicketService(s, ex)
	cfg := config.Config{BaseURL: "http://test"}
	h := NewAIHandler(ticketSvc, cfg)

	r := gin.New()
	r.POST("/api/v1/tickets", func(c *gin.Context) { c.Set("api_key_id", uint64(1)); c.Next() }, h.Submit)
	r.GET("/api/v1/tickets/:id", h.Get)
	return r, s
}

func TestAISubmitReturnsApprovalURL(t *testing.T) {
	r, _ := setupAI(t)
	body, _ := json.Marshal(map[string]any{
		"project": "demo", "name": "deploy", "args": map[string]any{"msg": "DRYRUN-OK"}})
	req := httptest.NewRequest("POST", "/api/v1/tickets", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp["approval_url"])
	assert.NotEmpty(t, resp["next_action"])
	assert.NotZero(t, resp["ticket_id"])
}

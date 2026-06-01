package api

import (
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
	"LoopGuard/internal/store"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type listTestEnv struct {
	r       *gin.Engine
	cfg     config.Config
	s       *store.Store
	appr    *model.User
	token   string
}

func setupListEnv(t *testing.T) *listTestEnv {
	gin.SetMode(gin.TestMode)
	s := testStore(t)

	appr := &model.User{Username: "appr", PasswordHash: "h", Role: model.RoleUser}
	require.NoError(t, s.CreateUser(appr))

	require.NoError(t, s.CreateProgram(&model.Program{
		Project: "infra", Name: "k8s-restart", EntryFile: "run.sh",
		Interpreter: "bash", ApproverID: appr.ID, TimeoutSec: 10,
		SupportsDryrun: true, Enabled: true,
	}))

	require.NoError(t, s.CreateAPIKey(&model.APIKey{
		Name: "prod-key", KeyHash: auth.HashAPIKey("x"), Enabled: true,
	}))

	ex := &fakeExecutor{result: &executor.ExecResult{
		ExitCode: 0, Stdout: "DRYRUN-OK", Stderr: "",
	}}
	ticketSvc := service.NewTicketService(s, ex, "/tmp/test-ws")
	cfg := config.Config{JWTSecret: "s"}

	h := NewHumanHandler(s, ticketSvc, cfg)
	r := gin.New()
	authGrp := r.Group("/", JWTAuth(cfg.JWTSecret))
	authGrp.GET("/tickets", h.ListMine)
	authGrp.GET("/tickets/:id", h.GetTicket)

	token, _ := auth.SignJWT(cfg.JWTSecret, appr.ID, string(appr.Role))

	return &listTestEnv{r: r, cfg: cfg, s: s, appr: appr, token: token}
}

func (e *listTestEnv) do(req *http.Request) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	return w
}

func TestListMineReturnsEnrichedFields(t *testing.T) {
	env := setupListEnv(t)
	require.NoError(t, env.s.CreateTicket(&model.Ticket{
		ProgramID: 1, SubmittedBy: 1, ApproverID: env.appr.ID,
		Status: model.StatusPendingApproval, Args: []byte(`["msg","DRYRUN-OK"]`),
	}))

	req := httptest.NewRequest("GET", "/tickets", nil)
	req.Header.Set("Authorization", "Bearer "+env.token)
	w := env.do(req)

	require.Equal(t, http.StatusOK, w.Code)

	var items []map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &items))
	require.Len(t, items, 1)

	assert.Equal(t, "infra", items[0]["program_project"])
	assert.Equal(t, "k8s-restart", items[0]["program_name"])
	assert.Equal(t, "prod-key", items[0]["submitted_by_name"])
}

func TestListMineOmitsLargeOutputs(t *testing.T) {
	env := setupListEnv(t)
	require.NoError(t, env.s.CreateTicket(&model.Ticket{
		ProgramID: 1, SubmittedBy: 1, ApproverID: env.appr.ID,
		Status: model.StatusPendingApproval, Args: []byte(`[]`),
		DryrunOutput: "big output", ExecOutput: "big exec",
	}))

	req := httptest.NewRequest("GET", "/tickets", nil)
	req.Header.Set("Authorization", "Bearer "+env.token)
	w := env.do(req)

	require.Equal(t, http.StatusOK, w.Code)

	var items []map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &items))
	require.Len(t, items, 1)

	_, hasDryrun := items[0]["dryrun_output"]
	_, hasExec := items[0]["exec_output"]
	assert.False(t, hasDryrun, "dryrun_output should not be in list response")
	assert.False(t, hasExec, "exec_output should not be in list response")
}

func TestListMineFallbackForDeletedProgram(t *testing.T) {
	env := setupListEnv(t)
	require.NoError(t, env.s.CreateTicket(&model.Ticket{
		ProgramID: 999, SubmittedBy: 1, ApproverID: env.appr.ID,
		Status: model.StatusPendingApproval, Args: []byte(`[]`),
	}))

	req := httptest.NewRequest("GET", "/tickets", nil)
	req.Header.Set("Authorization", "Bearer "+env.token)
	w := env.do(req)

	require.Equal(t, http.StatusOK, w.Code)

	var items []map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &items))
	require.Len(t, items, 1)

	assert.Equal(t, "", items[0]["program_project"])
	assert.Equal(t, "", items[0]["program_name"])
}

func TestListMineFallbackForDeletedAPIKey(t *testing.T) {
	env := setupListEnv(t)
	require.NoError(t, env.s.CreateTicket(&model.Ticket{
		ProgramID: 1, SubmittedBy: 888, ApproverID: env.appr.ID,
		Status: model.StatusPendingApproval, Args: []byte(`[]`),
	}))

	req := httptest.NewRequest("GET", "/tickets", nil)
	req.Header.Set("Authorization", "Bearer "+env.token)
	w := env.do(req)

	require.Equal(t, http.StatusOK, w.Code)

	var items []map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &items))
	require.Len(t, items, 1)

	assert.Equal(t, fmt.Sprintf("Key #%d", 888), items[0]["submitted_by_name"])
}

func TestGetTicketDetailStillReturnsFullModel(t *testing.T) {
	env := setupListEnv(t)
	require.NoError(t, env.s.CreateTicket(&model.Ticket{
		ProgramID: 1, SubmittedBy: 1, ApproverID: env.appr.ID,
		Status: model.StatusPendingApproval, Args: []byte(`[]`),
		DryrunOutput: "big output", ExecOutput: "big exec",
	}))

	req := httptest.NewRequest("GET", "/tickets/1", nil)
	req.Header.Set("Authorization", "Bearer "+env.token)
	w := env.do(req)

	require.Equal(t, http.StatusOK, w.Code)

	var detail map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &detail))
	assert.Equal(t, "big output", detail["dryrun_output"])
	assert.Equal(t, "big exec", detail["exec_output"])
}

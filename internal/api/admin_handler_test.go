package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"LoopGuard/internal/executor"
	"LoopGuard/internal/model"
	"LoopGuard/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdminCreateAPIKeyReturnsPlainOnce(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s := testStore(t)
	ex := executor.NewProcessExecutor()
	progSvc := service.NewProgramService(s, ex, "/tmp/test-ws")
	h := NewAdminHandler(s, progSvc, "/tmp/test-ws")

	r := gin.New()
	r.POST("/api-keys", h.CreateAPIKey)
	body := []byte(`{"name":"hermes-agent"}`)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, mustReq("POST", "/api-keys", body))

	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp["api_key"])
}

func TestAdminCreateUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s := testStore(t)
	h := NewAdminHandler(s, service.NewProgramService(s, executor.NewProcessExecutor(), "/tmp/test-ws"), "/tmp/test-ws")
	r := gin.New()
	r.POST("/users", h.CreateUser)
	body := []byte(`{"username":"bob","password":"pw123456","role":"user"}`)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, mustReq("POST", "/users", body))
	require.Equal(t, http.StatusOK, w.Code)

	u, err := s.GetUserByUsername("bob")
	require.NoError(t, err)
	assert.Equal(t, model.RoleUser, u.Role)
}

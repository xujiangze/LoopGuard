package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"LoopGuard/internal/config"
	"LoopGuard/internal/executor"
	"LoopGuard/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestRouterWiring(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s := testStore(t)
	ex := executor.NewProcessExecutor()
	deps := Deps{
		Store:      s,
		TicketSvc:  service.NewTicketService(s, ex),
		ProgramSvc: service.NewProgramService(s, ex),
		Cfg:        config.Config{JWTSecret: "s", BaseURL: "http://t"},
	}
	r := NewRouter(deps)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("POST", "/api/v1/tickets", nil))
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, httptest.NewRequest("GET", "/api/v1/tickets", nil))
	assert.Equal(t, http.StatusUnauthorized, w2.Code)
}

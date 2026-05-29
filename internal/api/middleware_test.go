package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"LoopGuard/internal/auth"
	"LoopGuard/internal/model"
	"LoopGuard/internal/store"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func testStore(t *testing.T) *store.Store {
	db, _ := gorm.Open(sqlite.Open(""), &gorm.Config{})
	s := store.New(db)
	require.NoError(t, s.AutoMigrate())
	return s
}

func TestAPIKeyMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s := testStore(t)
	plain := auth.GenerateAPIKey()
	require.NoError(t, s.CreateAPIKey(&model.APIKey{Name: "ai", KeyHash: auth.HashAPIKey(plain), Enabled: true}))

	r := gin.New()
	r.GET("/x", APIKeyAuth(s), func(c *gin.Context) { c.String(200, "ok") })

	req := httptest.NewRequest("GET", "/x", nil)
	req.Header.Set("X-API-Key", plain)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, httptest.NewRequest("GET", "/x", nil))
	assert.Equal(t, http.StatusUnauthorized, w2.Code)
}

func TestJWTMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	secret := "s"
	tok, _ := auth.SignJWT(secret, 5, "admin")

	r := gin.New()
	r.GET("/y", JWTAuth(secret), func(c *gin.Context) {
		uid := c.GetUint64("user_id")
		c.JSON(200, gin.H{"uid": uid})
	})
	req := httptest.NewRequest("GET", "/y", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
	assert.Contains(t, w.Body.String(), "5")
}

func TestAdminOnly(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/z", func(c *gin.Context) { c.Set("role", "user"); c.Next() }, AdminOnly(), func(c *gin.Context) { c.String(200, "ok") })
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/z", nil))
	assert.Equal(t, http.StatusForbidden, w.Code)
}

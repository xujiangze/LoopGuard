package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("LOOPGUARD_DB_DSN", "")
	cfg := Load()
	assert.Equal(t, ":8080", cfg.HTTPAddr)
	assert.Equal(t, "process", cfg.ExecutorType)
	assert.Equal(t, "./workspace", cfg.WorkspaceDir)
	assert.NotEmpty(t, cfg.JWTSecret)
}

func TestLoadFromEnv(t *testing.T) {
	t.Setenv("LOOPGUARD_HTTP_ADDR", ":9000")
	t.Setenv("LOOPGUARD_DB_DSN", "user:pass@tcp(localhost:3306)/lg")
	t.Setenv("LOOPGUARD_JWT_SECRET", "mysecret")
	cfg := Load()
	assert.Equal(t, ":9000", cfg.HTTPAddr)
	assert.Equal(t, "user:pass@tcp(localhost:3306)/lg", cfg.DBDSN)
	assert.Equal(t, "mysecret", cfg.JWTSecret)
}

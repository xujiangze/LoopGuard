package config

import "os"

type Config struct {
	HTTPAddr     string
	DBDSN        string
	JWTSecret    string
	ExecutorType string
	WorkspaceDir string
	BaseURL      string
}

func Load() Config {
	return Config{
		HTTPAddr:     env("LOOPGUARD_HTTP_ADDR", ":8080"),
		DBDSN:        env("LOOPGUARD_DB_DSN", ""),
		JWTSecret:    env("LOOPGUARD_JWT_SECRET", "dev-insecure-secret-change-me"),
		ExecutorType: env("LOOPGUARD_EXECUTOR_TYPE", "process"),
		WorkspaceDir: env("LOOPGUARD_WORKSPACE_DIR", "./workspace"),
		BaseURL:      env("LOOPGUARD_BASE_URL", "http://localhost:8080"),
	}
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

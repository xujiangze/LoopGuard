package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"LoopGuard/internal/executor"
	"LoopGuard/internal/model"
	"LoopGuard/internal/service"
	"LoopGuard/internal/store"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func versionTestSetup(t *testing.T) (*gin.Engine, *store.Store, string) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(""), &gorm.Config{})
	require.NoError(t, err)
	s := store.New(db)
	require.NoError(t, s.AutoMigrate())

	wsDir := t.TempDir()
	fe := &fakeExec{result: &executor.ExecResult{ExitCode: 0, Stdout: "usage: deploy [--only-print]"}}
	progSvc := service.NewProgramService(s, fe, wsDir)
	h := NewAdminHandler(s, progSvc, wsDir)
	r := gin.New()

	// 创建 admin 用户
	admin := &model.User{Username: "admin", PasswordHash: "hash", Role: model.RoleAdmin}
	require.NoError(t, s.CreateUser(admin))

	r.Use(func(c *gin.Context) { c.Set("userID", admin.ID); c.Set("role", string(admin.Role)) })
	r.GET("/programs/:id/files", h.ListFiles)
	r.GET("/programs/:id/files/:filename", h.GetFileContent)
	r.GET("/programs/:id/versions", h.ListVersions)
	r.GET("/programs/:id/versions/:version/files", h.ListVersionFiles)
	r.GET("/programs/:id/versions/:version/files/:filename", h.GetVersionFileContent)
	r.POST("/programs/:id/rollback", h.Rollback)
	r.DELETE("/programs/:id", h.DeleteProgram)

	// 注册一个带文件的程序
	prog := &model.Program{
		Project: "demo", Name: "deploy", EntryFile: "deploy.sh",
		Interpreter: "bash", ApproverID: admin.ID, TimeoutSec: 60,
		Enabled: true, CurrentVersion: 1, HelpText: "usage: deploy",
	}
	require.NoError(t, s.CreateProgram(prog))
	require.NoError(t, s.CreateProgramVersion(&model.ProgramVersion{
		ProgramID: prog.ID, Version: 1, EntryFile: "deploy.sh",
		Interpreter: "bash", HelpText: "usage: deploy", CreatedBy: "admin",
	}))

	// 创建磁盘文件
	progDir := filepath.Join(wsDir, "demo", "deploy")
	require.NoError(t, os.MkdirAll(progDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(progDir, "deploy.sh"), []byte("#!/bin/bash\necho deploy"), 0o644))
	snapDir := filepath.Join(wsDir, ".versions", fmt.Sprintf("%d", prog.ID), "v1")
	require.NoError(t, os.MkdirAll(snapDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(snapDir, "deploy.sh"), []byte("#!/bin/bash\necho v1"), 0o644))

	return r, s, wsDir
}

type fakeExec struct {
	result *executor.ExecResult
	err    error
}

func (f *fakeExec) Run(_ context.Context, _ executor.ExecRequest) (*executor.ExecResult, error) {
	return f.result, f.err
}

func TestListFiles(t *testing.T) {
	r, _, _ := versionTestSetup(t)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/programs/1/files", nil))
	require.Equal(t, http.StatusOK, w.Code)

	var files []service.FileInfo
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &files))
	require.Len(t, files, 1)
	require.Equal(t, "deploy.sh", files[0].Name)
	require.True(t, files[0].IsEntry)
}

func TestGetFileContent(t *testing.T) {
	r, _, _ := versionTestSetup(t)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/programs/1/files/deploy.sh", nil))
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "echo deploy")
}

func TestGetFileContentPathTraversal(t *testing.T) {
	r, _, _ := versionTestSetup(t)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/programs/1/files/..", nil))
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetFileContentNotFound(t *testing.T) {
	r, _, _ := versionTestSetup(t)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/programs/1/files/nonexist.sh", nil))
	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestListVersions(t *testing.T) {
	r, _, _ := versionTestSetup(t)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/programs/1/versions", nil))
	require.Equal(t, http.StatusOK, w.Code)

	var versions []model.ProgramVersion
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &versions))
	require.Len(t, versions, 1)
	require.Equal(t, 1, versions[0].Version)
}

func TestListVersionFiles(t *testing.T) {
	r, _, _ := versionTestSetup(t)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/programs/1/versions/1/files", nil))
	require.Equal(t, http.StatusOK, w.Code)

	var files []service.FileInfo
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &files))
	require.Len(t, files, 1)
	require.Equal(t, "deploy.sh", files[0].Name)
}

func TestGetVersionFileContent(t *testing.T) {
	r, _, _ := versionTestSetup(t)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/programs/1/versions/1/files/deploy.sh", nil))
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "echo v1")
}

func TestRollback(t *testing.T) {
	r, s, wsDir := versionTestSetup(t)

	// 先创建 v2
	p, _ := s.GetProgram(1)
	p.CurrentVersion = 2
	require.NoError(t, s.UpdateProgram(p))
	require.NoError(t, s.CreateProgramVersion(&model.ProgramVersion{
		ProgramID: 1, Version: 2, EntryFile: "deploy.sh", Interpreter: "bash", CreatedBy: "admin",
	}))
	snapDir := filepath.Join(wsDir, ".versions", "1", "v2")
	require.NoError(t, os.MkdirAll(snapDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(snapDir, "deploy.sh"), []byte("#!/bin/bash\necho v2"), 0o644))

	// 回滚到 v1
	body := []byte(`{"version":1}`)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, mustReq("POST", "/programs/1/rollback", body))
	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	// 版本应该变成 3（v1 -> v2 -> v3=rollback to v1）
	require.Equal(t, float64(3), resp["current_version"])
}

func TestDeleteProgram(t *testing.T) {
	r, _, wsDir := versionTestSetup(t)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, mustReq("DELETE", "/programs/1", nil))
	require.Equal(t, http.StatusOK, w.Code)

	// 磁盘文件已清理
	_, err := os.Stat(filepath.Join(wsDir, "demo", "deploy"))
	require.True(t, os.IsNotExist(err))
}

func TestDeleteProgramNotFound(t *testing.T) {
	r, _, _ := versionTestSetup(t)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, mustReq("DELETE", "/programs/999", nil))
	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestListFilesProgramNotFound(t *testing.T) {
	r, _, _ := versionTestSetup(t)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/programs/999/files", nil))
	require.Equal(t, http.StatusNotFound, w.Code)
}

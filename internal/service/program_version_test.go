package service

import (
	"bytes"
	"context"
	"mime/multipart"
	"os"
	"path/filepath"
	"testing"

	"LoopGuard/internal/executor"
	"LoopGuard/internal/model"
	"LoopGuard/internal/store"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newVersionTestSvc(t *testing.T, exec executor.Executor) (*ProgramService, *store.Store, string) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{})
	require.NoError(t, err)
	s := store.New(db)
	require.NoError(t, s.AutoMigrate())
	wsDir := t.TempDir()
	return NewProgramService(s, exec, wsDir), s, wsDir
}

// createFileHeader 创建真实的 multipart.FileHeader 用于测试
func createFileHeader(t *testing.T, filename, content string) *multipart.FileHeader {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, err := w.CreateFormFile("files", filename)
	require.NoError(t, err)
	_, err = part.Write([]byte(content))
	require.NoError(t, err)
	require.NoError(t, w.Close())

	reader := multipart.NewReader(&buf, w.Boundary())
	form, err := reader.ReadForm(10 << 20)
	require.NoError(t, err)
	return form.File["files"][0]
}

func TestRegisterCreatesVersionV1(t *testing.T) {
	fe := &fakeExecutor{result: &executor.ExecResult{ExitCode: 0, Stdout: "usage: deploy [--only-print]"}}
	svc, s, wsDir := newVersionTestSvc(t, fe)

	u := &model.User{Username: "appr", PasswordHash: "h", Role: model.RoleAdmin}
	require.NoError(t, s.CreateUser(u))

	p, err := svc.Register(context.Background(), RegisterInput{
		Project: "demo", Name: "deploy", EntryFile: "deploy.sh",
		Interpreter: "bash", ApproverID: u.ID, TimeoutSec: 60,
		Files: []*multipart.FileHeader{
			createFileHeader(t, "deploy.sh", "#!/bin/bash\necho deploy"),
		},
	})
	require.NoError(t, err)
	require.Equal(t, uint(1), p.CurrentVersion)

	// 验证 DB 中有 v1 记录
	pv, err := s.GetProgramVersion(p.ID, 1)
	require.NoError(t, err)
	require.Equal(t, "deploy.sh", pv.EntryFile)
	require.False(t, pv.IsRollback)

	// 验证磁盘上有快照
	snapshotDir := filepath.Join(wsDir, ".versions", "1", "v1")
	_, err = os.Stat(snapshotDir)
	require.NoError(t, err)
}

func TestUpdateWithFilesCreatesNewVersion(t *testing.T) {
	fe := &fakeExecutor{result: &executor.ExecResult{ExitCode: 0, Stdout: "usage: deploy [--only-print]"}}
	svc, s, wsDir := newVersionTestSvc(t, fe)

	u := &model.User{Username: "appr", PasswordHash: "h", Role: model.RoleAdmin}
	require.NoError(t, s.CreateUser(u))

	p, err := svc.Register(context.Background(), RegisterInput{
		Project: "demo", Name: "deploy", EntryFile: "deploy.sh",
		Interpreter: "bash", ApproverID: u.ID, TimeoutSec: 60,
		Files: []*multipart.FileHeader{
			createFileHeader(t, "deploy.sh", "#!/bin/bash\necho v1"),
		},
	})
	require.NoError(t, err)

	// 更新文件
	p, err = svc.Update(context.Background(), p.ID, UpdateInput{
		Files: []*multipart.FileHeader{
			createFileHeader(t, "deploy.sh", "#!/bin/bash\necho v2"),
		},
	})
	require.NoError(t, err)
	require.Equal(t, uint(2), p.CurrentVersion)

	// 验证 v2 存在
	pv2, err := s.GetProgramVersion(p.ID, 2)
	require.NoError(t, err)
	require.Equal(t, 2, pv2.Version)

	// 验证快照目录
	_, err = os.Stat(filepath.Join(wsDir, ".versions", "1", "v2"))
	require.NoError(t, err)
}

func TestUpdateWithoutFilesDoesNotCreateVersion(t *testing.T) {
	fe := &fakeExecutor{result: &executor.ExecResult{ExitCode: 0, Stdout: "usage: deploy [--only-print]"}}
	svc, s, _ := newVersionTestSvc(t, fe)

	u := &model.User{Username: "appr", PasswordHash: "h", Role: model.RoleAdmin}
	require.NoError(t, s.CreateUser(u))

	p, err := svc.Register(context.Background(), RegisterInput{
		Project: "demo", Name: "deploy", EntryFile: "deploy.sh",
		Interpreter: "bash", ApproverID: u.ID, TimeoutSec: 60,
		Files: []*multipart.FileHeader{
			createFileHeader(t, "deploy.sh", "#!/bin/bash\necho v1"),
		},
	})
	require.NoError(t, err)

	// 仅更新 enabled 状态
	p, err = svc.Update(context.Background(), p.ID, UpdateInput{
		Enabled: boolPtr(false),
	})
	require.NoError(t, err)
	require.Equal(t, uint(1), p.CurrentVersion)
}

func boolPtr(b bool) *bool { return &b }

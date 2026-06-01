package service

import (
	"context"
	"mime/multipart"
	"os"
	"path/filepath"
	"testing"

	"LoopGuard/internal/executor"
	"LoopGuard/internal/model"

	"github.com/stretchr/testify/require"
)

func TestListVersions(t *testing.T) {
	fe := &fakeExecutor{result: &executor.ExecResult{ExitCode: 0, Stdout: "usage: deploy [--only-print]"}}
	svc, s, _ := newVersionTestSvc(t, fe)

	u := &model.User{Username: "appr", PasswordHash: "h", Role: model.RoleAdmin}
	require.NoError(t, s.CreateUser(u))

	p, err := svc.Register(context.Background(), RegisterInput{
		Project: "demo", Name: "deploy", EntryFile: "deploy.sh",
		Interpreter: "bash", ApproverID: u.ID, TimeoutSec: 60,
		Files: []*multipart.FileHeader{createFileHeader(t, "deploy.sh", "v1")},
	})
	require.NoError(t, err)

	// 更新一次创建 v2
	_, err = svc.Update(context.Background(), p.ID, UpdateInput{
		Files: []*multipart.FileHeader{createFileHeader(t, "deploy.sh", "v2")},
	})
	require.NoError(t, err)

	versions, err := svc.ListVersions(p.ID)
	require.NoError(t, err)
	require.Len(t, versions, 2)
	require.Equal(t, 2, versions[0].Version) // 降序
	require.Equal(t, 1, versions[1].Version)
}

func TestGetCurrentFiles(t *testing.T) {
	fe := &fakeExecutor{result: &executor.ExecResult{ExitCode: 0, Stdout: "usage: deploy [--only-print]"}}
	svc, s, _ := newVersionTestSvc(t, fe)

	u := &model.User{Username: "appr", PasswordHash: "h", Role: model.RoleAdmin}
	require.NoError(t, s.CreateUser(u))

	p, err := svc.Register(context.Background(), RegisterInput{
		Project: "demo", Name: "deploy", EntryFile: "deploy.sh",
		Interpreter: "bash", ApproverID: u.ID, TimeoutSec: 60,
		Files: []*multipart.FileHeader{createFileHeader(t, "deploy.sh", "hello")},
	})
	require.NoError(t, err)

	files, err := svc.GetCurrentFiles(p)
	require.NoError(t, err)
	require.Len(t, files, 1)
	require.Equal(t, "deploy.sh", files[0].Name)
	require.True(t, files[0].IsEntry)
}

func TestGetCurrentFileContent(t *testing.T) {
	fe := &fakeExecutor{result: &executor.ExecResult{ExitCode: 0, Stdout: "usage: deploy [--only-print]"}}
	svc, s, _ := newVersionTestSvc(t, fe)

	u := &model.User{Username: "appr", PasswordHash: "h", Role: model.RoleAdmin}
	require.NoError(t, s.CreateUser(u))

	p, err := svc.Register(context.Background(), RegisterInput{
		Project: "demo", Name: "deploy", EntryFile: "deploy.sh",
		Interpreter: "bash", ApproverID: u.ID, TimeoutSec: 60,
		Files: []*multipart.FileHeader{createFileHeader(t, "deploy.sh", "#!/bin/bash\necho hello")},
	})
	require.NoError(t, err)

	content, err := svc.GetCurrentFileContent(p, "deploy.sh")
	require.NoError(t, err)
	require.Contains(t, content, "echo hello")
}

func TestGetCurrentFileContentRejectsPathTraversal(t *testing.T) {
	fe := &fakeExecutor{result: &executor.ExecResult{ExitCode: 0, Stdout: "usage: deploy [--only-print]"}}
	svc, s, _ := newVersionTestSvc(t, fe)

	u := &model.User{Username: "appr", PasswordHash: "h", Role: model.RoleAdmin}
	require.NoError(t, s.CreateUser(u))

	p, err := svc.Register(context.Background(), RegisterInput{
		Project: "demo", Name: "deploy", EntryFile: "deploy.sh",
		Interpreter: "bash", ApproverID: u.ID, TimeoutSec: 60,
		Files: []*multipart.FileHeader{createFileHeader(t, "deploy.sh", "v1")},
	})
	require.NoError(t, err)

	_, err = svc.GetCurrentFileContent(p, "../../etc/passwd")
	require.Error(t, err)
	require.Contains(t, err.Error(), "非法")
}

func TestGetVersionFiles(t *testing.T) {
	fe := &fakeExecutor{result: &executor.ExecResult{ExitCode: 0, Stdout: "usage: deploy [--only-print]"}}
	svc, s, _ := newVersionTestSvc(t, fe)

	u := &model.User{Username: "appr", PasswordHash: "h", Role: model.RoleAdmin}
	require.NoError(t, s.CreateUser(u))

	p, err := svc.Register(context.Background(), RegisterInput{
		Project: "demo", Name: "deploy", EntryFile: "deploy.sh",
		Interpreter: "bash", ApproverID: u.ID, TimeoutSec: 60,
		Files: []*multipart.FileHeader{createFileHeader(t, "deploy.sh", "v1")},
	})
	require.NoError(t, err)

	files, err := svc.GetVersionFiles(p.ID, 1)
	require.NoError(t, err)
	require.Len(t, files, 1)
	require.Equal(t, "deploy.sh", files[0].Name)
}

func TestGetVersionFileContent(t *testing.T) {
	fe := &fakeExecutor{result: &executor.ExecResult{ExitCode: 0, Stdout: "usage: deploy [--only-print]"}}
	svc, s, _ := newVersionTestSvc(t, fe)

	u := &model.User{Username: "appr", PasswordHash: "h", Role: model.RoleAdmin}
	require.NoError(t, s.CreateUser(u))

	p, err := svc.Register(context.Background(), RegisterInput{
		Project: "demo", Name: "deploy", EntryFile: "deploy.sh",
		Interpreter: "bash", ApproverID: u.ID, TimeoutSec: 60,
		Files: []*multipart.FileHeader{createFileHeader(t, "deploy.sh", "original content")},
	})
	require.NoError(t, err)

	content, err := svc.GetVersionFileContent(p.ID, 1, "deploy.sh")
	require.NoError(t, err)
	require.Contains(t, content, "original content")
}

func TestRollback(t *testing.T) {
	fe := &fakeExecutor{result: &executor.ExecResult{ExitCode: 0, Stdout: "usage: deploy [--only-print]"}}
	svc, s, wsDir := newVersionTestSvc(t, fe)

	u := &model.User{Username: "appr", PasswordHash: "h", Role: model.RoleAdmin}
	require.NoError(t, s.CreateUser(u))

	p, err := svc.Register(context.Background(), RegisterInput{
		Project: "demo", Name: "deploy", EntryFile: "deploy.sh",
		Interpreter: "bash", ApproverID: u.ID, TimeoutSec: 60,
		Files: []*multipart.FileHeader{createFileHeader(t, "deploy.sh", "v1 content")},
	})
	require.NoError(t, err)

	// 更新到 v2
	p, err = svc.Update(context.Background(), p.ID, UpdateInput{
		Files: []*multipart.FileHeader{createFileHeader(t, "deploy.sh", "v2 content")},
	})
	require.NoError(t, err)
	require.Equal(t, uint(2), p.CurrentVersion)

	// 回滚到 v1
	p, err = svc.Rollback(context.Background(), p.ID, 1, "admin")
	require.NoError(t, err)
	require.Equal(t, uint(3), p.CurrentVersion)

	// v3 应标记为 rollback
	pv3, err := s.GetProgramVersion(p.ID, 3)
	require.NoError(t, err)
	require.True(t, pv3.IsRollback)

	// 当前文件应该恢复为 v1 的内容
	currentContent, err := os.ReadFile(filepath.Join(wsDir, "demo", "deploy", "deploy.sh"))
	require.NoError(t, err)
	require.Contains(t, string(currentContent), "v1 content")
}

func TestRollbackNonExistentVersion(t *testing.T) {
	fe := &fakeExecutor{result: &executor.ExecResult{ExitCode: 0, Stdout: "usage: deploy [--only-print]"}}
	svc, s, _ := newVersionTestSvc(t, fe)

	u := &model.User{Username: "appr", PasswordHash: "h", Role: model.RoleAdmin}
	require.NoError(t, s.CreateUser(u))

	p, err := svc.Register(context.Background(), RegisterInput{
		Project: "demo", Name: "deploy", EntryFile: "deploy.sh",
		Interpreter: "bash", ApproverID: u.ID, TimeoutSec: 60,
		Files: []*multipart.FileHeader{createFileHeader(t, "deploy.sh", "v1")},
	})
	require.NoError(t, err)

	_, err = svc.Rollback(context.Background(), p.ID, 99, "admin")
	require.Error(t, err)
	require.Contains(t, err.Error(), "不存在")
}

func TestDeleteProgram(t *testing.T) {
	fe := &fakeExecutor{result: &executor.ExecResult{ExitCode: 0, Stdout: "usage: deploy [--only-print]"}}
	svc, s, wsDir := newVersionTestSvc(t, fe)

	u := &model.User{Username: "appr", PasswordHash: "h", Role: model.RoleAdmin}
	require.NoError(t, s.CreateUser(u))

	p, err := svc.Register(context.Background(), RegisterInput{
		Project: "demo", Name: "deploy", EntryFile: "deploy.sh",
		Interpreter: "bash", ApproverID: u.ID, TimeoutSec: 60,
		Files: []*multipart.FileHeader{createFileHeader(t, "deploy.sh", "v1")},
	})
	require.NoError(t, err)

	err = svc.DeleteProgram(p.ID)
	require.NoError(t, err)

	// 程序不存在
	_, err = s.GetProgram(p.ID)
	require.Error(t, err)

	// 磁盘文件已清理
	_, err = os.Stat(filepath.Join(wsDir, "demo", "deploy"))
	require.True(t, os.IsNotExist(err))
}

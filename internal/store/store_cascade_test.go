package store

import (
	"os"
	"path/filepath"
	"testing"

	"LoopGuard/internal/model"

	"github.com/stretchr/testify/require"
)

func TestDeleteProgramWithCascade(t *testing.T) {
	s := newTestDB(t)

	// 创建程序和版本记录
	p := &model.Program{
		Project: "demo", Name: "deploy", EntryFile: "deploy.sh",
		Interpreter: "bash", ApproverID: 1, TimeoutSec: 60, CurrentVersion: 2,
	}
	require.NoError(t, s.CreateProgram(p))
	require.NoError(t, s.CreateProgramVersion(&model.ProgramVersion{
		ProgramID: p.ID, Version: 1, EntryFile: "deploy.sh",
	}))
	require.NoError(t, s.CreateProgramVersion(&model.ProgramVersion{
		ProgramID: p.ID, Version: 2, EntryFile: "deploy.sh",
	}))

	// 创建模拟磁盘文件
	wsDir := t.TempDir()
	currentDir := filepath.Join(wsDir, "demo", "deploy")
	require.NoError(t, os.MkdirAll(currentDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(currentDir, "deploy.sh"), []byte("echo hello"), 0o644))
	versionsDir := filepath.Join(wsDir, ".versions", "1")
	require.NoError(t, os.MkdirAll(filepath.Join(versionsDir, "v1"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(versionsDir, "v2"), 0o755))

	err := s.DeleteProgramWithCascade(p.ID, wsDir)
	require.NoError(t, err)

	// 验证 DB 记录已删除
	_, err = s.GetProgram(p.ID)
	require.Error(t, err) // record not found

	versions, err := s.ListProgramVersions(p.ID)
	require.NoError(t, err)
	require.Empty(t, versions)

	// 验证磁盘文件已清理
	_, err = os.Stat(currentDir)
	require.True(t, os.IsNotExist(err))
	_, err = os.Stat(versionsDir)
	require.True(t, os.IsNotExist(err))
}

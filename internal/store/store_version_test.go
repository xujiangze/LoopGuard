package store

import (
	"testing"

	"LoopGuard/internal/model"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newTestDB(t *testing.T) *Store {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{})
	require.NoError(t, err)
	s := New(db)
	require.NoError(t, s.AutoMigrate())
	return s
}

func TestCreateAndGetProgramVersion(t *testing.T) {
	s := newTestDB(t)

	p := &model.Program{
		Project: "demo", Name: "deploy", EntryFile: "deploy.sh",
		Interpreter: "bash", ApproverID: 1, TimeoutSec: 60,
		CurrentVersion: 1,
	}
	require.NoError(t, s.CreateProgram(p))
	require.Equal(t, uint(1), p.CurrentVersion)

	pv := &model.ProgramVersion{
		ProgramID:  p.ID,
		Version:    1,
		EntryFile:  "deploy.sh",
		Interpreter: "bash",
		HelpText:   "usage: deploy [--only-print]",
		CreatedBy:  "admin",
	}
	require.NoError(t, s.CreateProgramVersion(pv))
	require.NotZero(t, pv.ID)

	got, err := s.GetProgramVersion(p.ID, 1)
	require.NoError(t, err)
	require.Equal(t, "deploy.sh", got.EntryFile)
	require.False(t, got.IsRollback)
}

func TestListProgramVersions(t *testing.T) {
	s := newTestDB(t)

	p := &model.Program{
		Project: "p", Name: "n", EntryFile: "main.sh",
		Interpreter: "bash", ApproverID: 1, TimeoutSec: 60,
	}
	require.NoError(t, s.CreateProgram(p))

	for i := 1; i <= 3; i++ {
		require.NoError(t, s.CreateProgramVersion(&model.ProgramVersion{
			ProgramID: p.ID, Version: i, EntryFile: "main.sh",
		}))
	}

	versions, err := s.ListProgramVersions(p.ID)
	require.NoError(t, err)
	require.Len(t, versions, 3)
	// 降序排列
	require.Equal(t, 3, versions[0].Version)
	require.Equal(t, 1, versions[2].Version)
}

func TestDeleteProgramVersionsByProgramID(t *testing.T) {
	s := newTestDB(t)

	p := &model.Program{Project: "p", Name: "n", EntryFile: "m.sh", Interpreter: "bash", ApproverID: 1}
	require.NoError(t, s.CreateProgram(p))
	require.NoError(t, s.CreateProgramVersion(&model.ProgramVersion{ProgramID: p.ID, Version: 1}))
	require.NoError(t, s.CreateProgramVersion(&model.ProgramVersion{ProgramID: p.ID, Version: 2}))

	require.NoError(t, s.DeleteProgramVersionsByProgramID(p.ID))

	versions, err := s.ListProgramVersions(p.ID)
	require.NoError(t, err)
	require.Empty(t, versions)
}

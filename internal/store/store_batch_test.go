package store

import (
	"testing"

	"LoopGuard/internal/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetProgramsByIDs(t *testing.T) {
	s := newTestStore(t)
	u := &model.User{Username: "appr", PasswordHash: "h", Role: model.RoleUser}
	require.NoError(t, s.CreateUser(u))

	p1 := &model.Program{Project: "infra", Name: "k8s-restart", EntryFile: "run.sh", Interpreter: "bash", ApproverID: u.ID, Enabled: true}
	p2 := &model.Program{Project: "app", Name: "deploy", EntryFile: "deploy.sh", Interpreter: "bash", ApproverID: u.ID, Enabled: true}
	require.NoError(t, s.CreateProgram(p1))
	require.NoError(t, s.CreateProgram(p2))

	m, err := s.GetProgramsByIDs([]uint64{p1.ID, p2.ID, 999})
	require.NoError(t, err)
	assert.Len(t, m, 2)
	assert.Equal(t, "infra", m[p1.ID].Project)
	assert.Equal(t, "app", m[p2.ID].Project)
}

func TestGetProgramsByIDsEmpty(t *testing.T) {
	s := newTestStore(t)
	m, err := s.GetProgramsByIDs(nil)
	require.NoError(t, err)
	assert.Empty(t, m)
}

func TestGetAPIKeysByIDs(t *testing.T) {
	s := newTestStore(t)
	k1 := &model.APIKey{Name: "prod-key", KeyHash: "h1", Enabled: true}
	k2 := &model.APIKey{Name: "dev-key", KeyHash: "h2", Enabled: true}
	require.NoError(t, s.CreateAPIKey(k1))
	require.NoError(t, s.CreateAPIKey(k2))

	m, err := s.GetAPIKeysByIDs([]uint64{k1.ID, k2.ID, 999})
	require.NoError(t, err)
	assert.Len(t, m, 2)
	assert.Equal(t, "prod-key", m[k1.ID].Name)
	assert.Equal(t, "dev-key", m[k2.ID].Name)
}

func TestGetAPIKeysByIDsEmpty(t *testing.T) {
	s := newTestStore(t)
	m, err := s.GetAPIKeysByIDs(nil)
	require.NoError(t, err)
	assert.Empty(t, m)
}

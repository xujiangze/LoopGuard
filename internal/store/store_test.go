package store

import (
	"testing"

	"LoopGuard/internal/model"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newTestStore(t *testing.T) *Store {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	s := New(db)
	require.NoError(t, s.AutoMigrate())
	return s
}

func TestCreateAndGetUser(t *testing.T) {
	s := newTestStore(t)
	u := &model.User{Username: "alice", PasswordHash: "h", Role: model.RoleAdmin}
	require.NoError(t, s.CreateUser(u))
	require.NotZero(t, u.ID)

	got, err := s.GetUserByUsername("alice")
	require.NoError(t, err)
	require.Equal(t, model.RoleAdmin, got.Role)
}

func TestGetProgramByProjectName(t *testing.T) {
	s := newTestStore(t)
	u := &model.User{Username: "approver", PasswordHash: "h", Role: model.RoleUser}
	require.NoError(t, s.CreateUser(u))
	p := &model.Program{Project: "demo", Name: "deploy", BinaryPath: "/bin/x", ApproverID: u.ID, Enabled: true}
	require.NoError(t, s.CreateProgram(p))

	got, err := s.GetProgramByProjectName("demo", "deploy")
	require.NoError(t, err)
	require.Equal(t, p.ID, got.ID)
}

func TestTicketLifecycle(t *testing.T) {
	s := newTestStore(t)
	tk := &model.Ticket{ProgramID: 1, ApproverID: 1, SubmittedBy: 1,
		Status: model.StatusPendingDryrun, Args: []byte(`{}`)}
	require.NoError(t, s.CreateTicket(tk))
	require.NotZero(t, tk.ID)

	tk.Status = model.StatusPendingApproval
	require.NoError(t, s.UpdateTicket(tk))

	got, err := s.GetTicket(tk.ID)
	require.NoError(t, err)
	require.Equal(t, model.StatusPendingApproval, got.Status)
}

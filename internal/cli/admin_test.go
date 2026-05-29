package cli

import (
	"testing"

	"LoopGuard/internal/auth"
	"LoopGuard/internal/store"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestCreateUserLogic(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	s := store.New(db)
	require.NoError(t, s.AutoMigrate())

	err := CreateUserInStore(s, "root", "rootpw123", true)
	require.NoError(t, err)

	u, err := s.GetUserByUsername("root")
	require.NoError(t, err)
	assert.Equal(t, "admin", string(u.Role))
	assert.True(t, auth.VerifyPassword(u.PasswordHash, "rootpw123"))
}

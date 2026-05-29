package model

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestAutoMigrate(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	err = db.AutoMigrate(&User{}, &APIKey{}, &Program{}, &Ticket{}, &Execution{})
	require.NoError(t, err)

	require.True(t, db.Migrator().HasTable(&User{}))
	require.True(t, db.Migrator().HasTable(&Ticket{}))
	require.True(t, db.Migrator().HasColumn(&Ticket{}, "dryrun_output"))
	require.True(t, db.Migrator().HasColumn(&Program{}, "supports_dryrun"))
}

package cli

import (
	"LoopGuard/internal/config"
	"LoopGuard/internal/store"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func openStore(cfg config.Config) (*store.Store, error) {
	db, err := gorm.Open(mysql.Open(cfg.DBDSN), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	return store.New(db), nil
}

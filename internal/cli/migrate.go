package cli

import (
	"fmt"

	"LoopGuard/internal/config"

	"github.com/spf13/cobra"
)

func MigrateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "migrate",
		Short: "执行数据库迁移（AutoMigrate）",
		RunE: func(c *cobra.Command, _ []string) error {
			s, err := openStore(config.Load())
			if err != nil {
				return err
			}
			if err := s.AutoMigrate(); err != nil {
				return err
			}
			fmt.Println("迁移完成")
			return nil
		},
	}
}

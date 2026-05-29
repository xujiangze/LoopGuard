package cli

import (
	"fmt"

	"LoopGuard/internal/api"
	"LoopGuard/internal/config"
	"LoopGuard/internal/executor"
	"LoopGuard/internal/service"

	"github.com/spf13/cobra"
)

func ServeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "启动 HTTP 服务",
		RunE: func(c *cobra.Command, _ []string) error {
			cfg := config.Load()
			s, err := openStore(cfg)
			if err != nil {
				return err
			}
			if err := s.AutoMigrate(); err != nil {
				return err
			}
			var ex executor.Executor = executor.NewProcessExecutor()
			deps := api.Deps{
				Store:      s,
				TicketSvc:  service.NewTicketService(s, ex),
				ProgramSvc: service.NewProgramService(s, ex),
				Cfg:        cfg,
			}
			r := api.NewRouter(deps)
			fmt.Printf("LoopGuard 监听 %s\n", cfg.HTTPAddr)
			return r.Run(cfg.HTTPAddr)
		},
	}
}

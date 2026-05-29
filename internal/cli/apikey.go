package cli

import (
	"fmt"

	"LoopGuard/internal/auth"
	"LoopGuard/internal/config"
	"LoopGuard/internal/model"

	"github.com/spf13/cobra"
)

func APIKeyCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "apikey", Short: "AI 服务账号 Key 管理"}
	var name string
	create := &cobra.Command{
		Use:   "create",
		Short: "创建 API Key（明文仅打印一次）",
		RunE: func(c *cobra.Command, _ []string) error {
			s, err := openStore(config.Load())
			if err != nil {
				return err
			}
			plain := auth.GenerateAPIKey()
			k := &model.APIKey{Name: name, KeyHash: auth.HashAPIKey(plain), Enabled: true}
			if err := s.CreateAPIKey(k); err != nil {
				return err
			}
			fmt.Printf("API Key 已创建（请妥善保存，只显示这一次）：\n  name: %s\n  key:  %s\n", name, plain)
			return nil
		},
	}
	create.Flags().StringVar(&name, "name", "", "Key 名称，如 hermes-agent")
	_ = create.MarkFlagRequired("name")
	cmd.AddCommand(create)
	return cmd
}

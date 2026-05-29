package cli

import (
	"fmt"

	"LoopGuard/internal/auth"
	"LoopGuard/internal/config"
	"LoopGuard/internal/model"
	"LoopGuard/internal/store"

	"github.com/spf13/cobra"
)

func CreateUserInStore(s *store.Store, username, password string, admin bool) error {
	hash, err := auth.HashPassword(password)
	if err != nil {
		return err
	}
	role := model.RoleUser
	if admin {
		role = model.RoleAdmin
	}
	return s.CreateUser(&model.User{Username: username, PasswordHash: hash, Role: role})
}

func AdminCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "admin", Short: "用户管理"}
	var username, password string
	var isAdmin bool
	create := &cobra.Command{
		Use:   "create-user",
		Short: "创建用户（首个 admin 用 --admin）",
		RunE: func(c *cobra.Command, _ []string) error {
			s, err := openStore(config.Load())
			if err != nil {
				return err
			}
			if err := CreateUserInStore(s, username, password, isAdmin); err != nil {
				return err
			}
			fmt.Printf("已创建用户 %s（admin=%v）\n", username, isAdmin)
			return nil
		},
	}
	create.Flags().StringVar(&username, "username", "", "用户名")
	create.Flags().StringVar(&password, "password", "", "密码")
	create.Flags().BoolVar(&isAdmin, "admin", false, "设为管理员")
	_ = create.MarkFlagRequired("username")
	_ = create.MarkFlagRequired("password")
	cmd.AddCommand(create)
	return cmd
}

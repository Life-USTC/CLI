package me

import (
	"github.com/spf13/cobra"

	"github.com/Life-USTC/CLI/internal/api"
	"github.com/Life-USTC/CLI/internal/cmd/cmdutil"
	"github.com/Life-USTC/CLI/internal/output"
)

func NewCmdMe() *cobra.Command {
	return &cobra.Command{
		Use:   "me",
		Short: "Show your profile",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := api.NewClient(cmdutil.ServerFromCmd(cmd), true)
			if err != nil {
				return err
			}
			data, err := c.Get("/api/me", nil)
			if err != nil {
				return err
			}
			output.OutputDetail(data, []output.FieldDef{
				{Key: "id", Label: "ID"},
				{Key: "name", Label: "Name"},
				{Key: "email", Label: "Email"},
				{Key: "username", Label: "Username"},
				{Key: "isAdmin", Label: "Admin"},
			}, "Profile")
			return nil
		},
	}
}

package me

import (
	"github.com/spf13/cobra"

	"github.com/Life-USTC/CLI/internal/api"
	"github.com/Life-USTC/CLI/internal/cmd/calendar"
	"github.com/Life-USTC/CLI/internal/cmd/cmdutil"
	"github.com/Life-USTC/CLI/internal/cmd/homework"
	"github.com/Life-USTC/CLI/internal/cmd/todo"
	"github.com/Life-USTC/CLI/internal/cmd/upload"
	"github.com/Life-USTC/CLI/internal/output"
)

func NewCmdMe() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "me [command]",
		Short: "Your personal hub",
		Long:  "Show your profile or manage personal data (todos, homework, calendar, uploads).",
		Example: `  # Show your profile
  life-ustc me

  # List your pending homeworks
  life-ustc me homework list --pending

  # Manage todos
  life-ustc me todo list --done`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := api.NewTypedClient(cmdutil.ServerFromCmd(cmd), true)
			if err != nil {
				return err
			}
			data, err := api.ParseResponseRaw(c.GetMe(api.Ctx()))
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

	cmd.AddCommand(todo.NewCmdTodo())
	cmd.AddCommand(homework.NewCmdMyHomework())
	cmd.AddCommand(calendar.NewCmdCalendar())
	cmd.AddCommand(upload.NewCmdUpload())

	return cmd
}

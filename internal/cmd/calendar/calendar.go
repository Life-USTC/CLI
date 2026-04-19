package calendar

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Life-USTC/CLI/internal/api"
	"github.com/Life-USTC/CLI/internal/cmd/cmdutil"
	"github.com/Life-USTC/CLI/internal/output"
)

func NewCmdCalendar() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "calendar [command]",
		Short: "Manage calendar subscriptions",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCalendarGet(cmd)
		},
	}
	cmd.AddCommand(newCmdGet())
	cmd.AddCommand(newCmdSet())
	return cmd
}

func runCalendarGet(cmd *cobra.Command) error {
	c, err := api.NewClient(cmdutil.ServerFromCmd(cmd), true)
	if err != nil {
		return err
	}
	data, err := c.Get("/api/calendar-subscriptions/current", nil)
	if err != nil {
		return err
	}
	if output.IsJSON() {
		output.JSON(data)
		return nil
	}
	m := cmdutil.AsMap(data)
	output.KVWithTitle([]output.KVPair{
		{Key: "URL", Value: output.Resolve(m, "url")},
		{Key: "Note", Value: output.Resolve(m, "note")},
	}, "Calendar subscription")

	if sections, ok := m["sections"].([]any); ok && len(sections) > 0 {
		fmt.Println()
		output.Bold("  Sections")
		var rows []map[string]any
		for _, s := range sections {
			if row, ok := s.(map[string]any); ok {
				rows = append(rows, row)
			}
		}
		output.Table(rows, []output.Column{
			{Header: "Code", Key: "code"},
			{Header: "Course", Key: "course.namePrimary"},
			{Header: "Semester", Key: "semester.name"},
		})
	}
	return nil
}

func newCmdGet() *cobra.Command {
	return &cobra.Command{
		Use:   "get",
		Short: "Show your calendar subscription",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCalendarGet(cmd)
		},
	}
}

func newCmdSet() *cobra.Command {
	return &cobra.Command{
		Use:   "set <section-id>...",
		Short: "Set calendar section subscriptions",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := api.NewClient(cmdutil.ServerFromCmd(cmd), true)
			if err != nil {
				return err
			}
			_, err = c.Post("/api/calendar-subscriptions", map[string]any{
				"sectionIds": strings.Join(args, ","),
			})
			if err != nil {
				return err
			}
			output.Success("Calendar subscriptions updated.")
			return nil
		},
	}
}

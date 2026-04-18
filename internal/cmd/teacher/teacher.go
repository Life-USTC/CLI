package teacher

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/Life-USTC/CLI/internal/api"
	"github.com/Life-USTC/CLI/internal/cmd/cmdutil"
	"github.com/Life-USTC/CLI/internal/output"
)

func NewCmdTeacher() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "teacher <command>",
		Short: "Browse teachers",
	}
	cmd.AddCommand(newCmdList())
	cmd.AddCommand(newCmdView())
	return cmd
}

func newCmdList() *cobra.Command {
	var (
		departmentID string
		search       string
		page, limit  int
	)
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List teachers",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := api.NewClient(cmdutil.ServerFromCmd(cmd), false)
			if err != nil {
				return err
			}
			params := url.Values{}
			if departmentID != "" {
				params.Set("departmentId", departmentID)
			}
			if search != "" {
				params.Set("search", search)
			}
			if page > 0 {
				params.Set("page", cmdutil.Itoa(page))
			}
			if limit > 0 {
				params.Set("limit", cmdutil.Itoa(limit))
			}
			data, err := c.Get("/api/teachers", params)
			if err != nil {
				return err
			}
			raw, rows, total, pg := cmdutil.ExtractList(data)
			output.OutputList(raw, rows, []output.Column{
				{Header: "Code", Key: "code"},
				{Header: "Name", Key: "namePrimary"},
				{Header: "Name (EN)", Key: "nameSecondary"},
				{Header: "Department", Key: "department.name"},
			}, total, pg)
			return nil
		},
	}
	cmd.Flags().StringVar(&departmentID, "department-id", "", "Department ID")
	cmd.Flags().StringVarP(&search, "search", "s", "", "Search query")
	cmd.Flags().IntVar(&page, "page", 0, "Page number")
	cmd.Flags().IntVar(&limit, "limit", 0, "Results per page")
	return cmd
}

func newCmdView() *cobra.Command {
	return &cobra.Command{
		Use:   "view <teacher-id>",
		Short: "View teacher details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := api.NewClient(cmdutil.ServerFromCmd(cmd), false)
			if err != nil {
				return err
			}
			data, err := c.Get(fmt.Sprintf("/api/teachers/%s", args[0]), nil)
			if err != nil {
				return err
			}
			if output.IsJSON() {
				output.JSON(data)
				return nil
			}
			m := cmdutil.AsMap(data)
			output.KVWithTitle([]output.KVPair{
				{Key: "Code", Value: output.Resolve(m, "code")},
				{Key: "Name", Value: output.Resolve(m, "namePrimary")},
				{Key: "Name (EN)", Value: output.Resolve(m, "nameSecondary")},
				{Key: "Department", Value: output.Resolve(m, "department.name")},
				{Key: "Title", Value: output.Resolve(m, "title")},
			}, "Teacher")

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
		},
	}
}

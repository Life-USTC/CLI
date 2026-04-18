package course

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/Life-USTC/CLI/internal/api"
	"github.com/Life-USTC/CLI/internal/cmd/cmdutil"
	"github.com/Life-USTC/CLI/internal/output"
)

func NewCmdCourse() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "course <command>",
		Short: "Browse courses",
	}
	cmd.AddCommand(newCmdList())
	cmd.AddCommand(newCmdView())
	return cmd
}

func newCmdList() *cobra.Command {
	var (
		search           string
		educationLevelID string
		categoryID       string
		page, limit      int
	)
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List courses",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := api.NewClient(cmdutil.ServerFromCmd(cmd), false)
			if err != nil {
				return err
			}
			params := url.Values{}
			if search != "" {
				params.Set("search", search)
			}
			if educationLevelID != "" {
				params.Set("educationLevelId", educationLevelID)
			}
			if categoryID != "" {
				params.Set("categoryId", categoryID)
			}
			if page > 0 {
				params.Set("page", cmdutil.Itoa(page))
			}
			if limit > 0 {
				params.Set("limit", cmdutil.Itoa(limit))
			}
			data, err := c.Get("/api/courses", params)
			if err != nil {
				return err
			}
			raw, rows, total, pg := cmdutil.ExtractList(data)
			output.OutputList(raw, rows, []output.Column{
				{Header: "Code", Key: "code"},
				{Header: "Name", Key: "namePrimary"},
				{Header: "Name (EN)", Key: "nameSecondary"},
				{Header: "Level", Key: "educationLevel.name"},
			}, total, pg)
			return nil
		},
	}
	cmd.Flags().StringVarP(&search, "search", "s", "", "Search query")
	cmd.Flags().StringVar(&educationLevelID, "education-level-id", "", "Education level ID")
	cmd.Flags().StringVar(&categoryID, "category-id", "", "Category ID")
	cmd.Flags().IntVar(&page, "page", 0, "Page number")
	cmd.Flags().IntVar(&limit, "limit", 0, "Results per page")
	return cmd
}

func newCmdView() *cobra.Command {
	return &cobra.Command{
		Use:   "view <jw-id>",
		Short: "View course details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := api.NewClient(cmdutil.ServerFromCmd(cmd), false)
			if err != nil {
				return err
			}
			data, err := c.Get(fmt.Sprintf("/api/courses/%s", args[0]), nil)
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
				{Key: "Level", Value: output.Resolve(m, "educationLevel.name")},
				{Key: "Category", Value: output.Resolve(m, "category.name")},
				{Key: "Class type", Value: output.Resolve(m, "classType.name")},
				{Key: "Gradation", Value: output.Resolve(m, "gradation.name")},
				{Key: "Course type", Value: output.Resolve(m, "courseType.name")},
			}, "Course")

			// Sections sub-table
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
					{Header: "Semester", Key: "semester.name"},
					{Header: "Campus", Key: "campus.name"},
				})
			}
			return nil
		},
	}
}

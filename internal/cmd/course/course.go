package course

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/Life-USTC/CLI/internal/api"
	"github.com/Life-USTC/CLI/internal/cmd/cmdutil"
	"github.com/Life-USTC/CLI/internal/cmd/comment"
	"github.com/Life-USTC/CLI/internal/cmd/description"
	"github.com/Life-USTC/CLI/internal/output"
)

func NewCmdCourse() *cobra.Command {
	var opts courseListOpts
	cmd := &cobra.Command{
		Use:   "course [command]",
		Short: "Browse courses",
		Long:  "List and view courses offered at USTC.",
		Example: `  # List all courses
  life-ustc course

  # Search courses by keyword
  life-ustc course -s "linear algebra"

  # View a specific course
  life-ustc course view <jw-id>`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCourseList(cmd, opts)
		},
	}
	addCourseListFlags(cmd, &opts)
	cmd.AddCommand(newCmdList())
	cmd.AddCommand(newCmdView())
	cmd.AddCommand(comment.NewCmdCommentFor("course"))
	cmd.AddCommand(description.NewCmdDescriptionFor("course"))
	return cmd
}

type courseListOpts struct {
	search           string
	educationLevelID string
	categoryID       string
	page             int
	limit            int
}

func runCourseList(cmd *cobra.Command, opts courseListOpts) error {
	c, err := api.NewClient(cmdutil.ServerFromCmd(cmd), false)
	if err != nil {
		return err
	}
	params := url.Values{}
	if opts.search != "" {
		params.Set("search", opts.search)
	}
	if opts.educationLevelID != "" {
		params.Set("educationLevelId", opts.educationLevelID)
	}
	if opts.categoryID != "" {
		params.Set("categoryId", opts.categoryID)
	}
	cmdutil.ApplyListParams(params, opts.page, opts.limit)
	data, err := c.Get("/api/courses", params)
	if err != nil {
		return err
	}
	raw, rows, total, pg := cmdutil.ExtractList(data)
	output.OutputList(raw, rows, []output.Column{
		{Header: "ID", Key: "id"},
		{Header: "Code", Key: "code"},
		{Header: "Name", Key: "namePrimary"},
		{Header: "Name (EN)", Key: "nameSecondary"},
		{Header: "Level", Key: "educationLevel.name"},
	}, total, pg)
	return nil
}

func addCourseListFlags(cmd *cobra.Command, opts *courseListOpts) {
	cmd.Flags().StringVarP(&opts.search, "search", "s", "", "Search query")
	cmd.Flags().StringVar(&opts.educationLevelID, "education-level-id", "", "Education level ID")
	cmd.Flags().StringVar(&opts.categoryID, "category-id", "", "Category ID")
	cmdutil.AddListFlags(cmd, &opts.page, &opts.limit)
}

func newCmdList() *cobra.Command {
	var opts courseListOpts
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List courses",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCourseList(cmd, opts)
		},
	}
	addCourseListFlags(cmd, &opts)
	return cmd
}

func newCmdView() *cobra.Command {
	return &cobra.Command{
		Use:     "view <jw-id>",
		Aliases: []string{"show"},
		Short:   "View course details",
		Args:    cobra.ExactArgs(1),
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
				{Key: "ID", Value: output.Resolve(m, "id")},
				{Key: "Code", Value: output.Resolve(m, "code")},
				{Key: "Name", Value: output.Resolve(m, "namePrimary")},
				{Key: "Name (EN)", Value: output.Resolve(m, "nameSecondary")},
				{Key: "Level", Value: output.Resolve(m, "educationLevel.name")},
				{Key: "Category", Value: output.Resolve(m, "category.name")},
				{Key: "Class type", Value: output.Resolve(m, "classType.name")},
				{Key: "Gradation", Value: output.Resolve(m, "gradation.name")},
				{Key: "Course type", Value: output.Resolve(m, "courseType.name")},
			}, "Course")

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
					{Header: "ID", Key: "id"},
					{Header: "Code", Key: "code"},
					{Header: "Semester", Key: "semester.name"},
					{Header: "Campus", Key: "campus.name"},
				})
			}
			return nil
		},
	}
}

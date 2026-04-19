package teacher

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Life-USTC/CLI/internal/api"
	"github.com/Life-USTC/CLI/internal/cmd/cmdutil"
	"github.com/Life-USTC/CLI/internal/cmd/comment"
	"github.com/Life-USTC/CLI/internal/cmd/description"
	openapi "github.com/Life-USTC/CLI/internal/openapi"
	"github.com/Life-USTC/CLI/internal/output"
)

func NewCmdTeacher() *cobra.Command {
	var opts teacherListOpts
	cmd := &cobra.Command{
		Use:   "teacher [command]",
		Short: "Browse teachers",
		Long:  "List and view teacher profiles and their associated sections.",
		Example: `  # List all teachers
  life-ustc teacher

  # Search teachers by name
  life-ustc teacher -s "zhang"

  # View a specific teacher
  life-ustc teacher view <teacher-id>`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTeacherList(cmd, opts)
		},
	}
	addTeacherListFlags(cmd, &opts)
	cmd.AddCommand(newCmdList())
	cmd.AddCommand(newCmdView())
	cmd.AddCommand(comment.NewCmdCommentFor("teacher"))
	cmd.AddCommand(description.NewCmdDescriptionFor("teacher"))
	return cmd
}

type teacherListOpts struct {
	departmentID string
	search       string
	page         int
	limit        int
}

func runTeacherList(cmd *cobra.Command, opts teacherListOpts) error {
	c, err := api.NewTypedClient(cmdutil.ServerFromCmd(cmd), false)
	if err != nil {
		return err
	}
	params := openapi.ListTeachersParams{}
	if opts.departmentID != "" {
		params.DepartmentId = &opts.departmentID
	}
	if opts.search != "" {
		params.Search = &opts.search
	}
	if opts.page > 0 {
		p := cmdutil.Itoa(opts.page)
		params.Page = &p
	}
	if opts.limit > 0 {
		l := cmdutil.Itoa(opts.limit)
		params.Limit = &l
	}
	data, err := api.ParseResponseRaw(c.ListTeachers(api.Ctx(), &params))
	if err != nil {
		return err
	}
	raw, rows, total, pg := cmdutil.ExtractList(data)
	output.OutputList(raw, rows, []output.Column{
		{Header: "ID", Key: "id"},
		{Header: "Code", Key: "code"},
		{Header: "Name", Key: "namePrimary"},
		{Header: "Name (EN)", Key: "nameSecondary"},
		{Header: "Department", Key: "department.name"},
	}, total, pg)
	return nil
}

func addTeacherListFlags(cmd *cobra.Command, opts *teacherListOpts) {
	cmd.Flags().StringVar(&opts.departmentID, "department-id", "", "Department ID")
	cmd.Flags().StringVarP(&opts.search, "search", "s", "", "Search query")
	cmdutil.AddListFlags(cmd, &opts.page, &opts.limit)
}

func newCmdList() *cobra.Command {
	var opts teacherListOpts
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List teachers",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTeacherList(cmd, opts)
		},
	}
	addTeacherListFlags(cmd, &opts)
	return cmd
}

func newCmdView() *cobra.Command {
	return &cobra.Command{
		Use:     "view <teacher-id>",
		Aliases: []string{"show"},
		Short:   "View teacher details",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := api.NewTypedClient(cmdutil.ServerFromCmd(cmd), false)
			if err != nil {
				return err
			}
			data, err := api.ParseResponseRaw(c.GetTeacher(api.Ctx(), args[0]))
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
					{Header: "ID", Key: "id"},
					{Header: "Code", Key: "code"},
					{Header: "Course", Key: "course.namePrimary"},
					{Header: "Semester", Key: "semester.name"},
				})
			}
			return nil
		},
	}
}

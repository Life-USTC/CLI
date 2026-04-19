package section

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/Life-USTC/CLI/internal/api"
	"github.com/Life-USTC/CLI/internal/cmd/cmdutil"
	"github.com/Life-USTC/CLI/internal/cmd/comment"
	"github.com/Life-USTC/CLI/internal/cmd/description"
	"github.com/Life-USTC/CLI/internal/cmd/homework"
	openapi "github.com/Life-USTC/CLI/internal/openapi"
	"github.com/Life-USTC/CLI/internal/output"
)

func NewCmdSection() *cobra.Command {
	var opts sectionListOpts
	cmd := &cobra.Command{
		Use:   "section [command]",
		Short: "Browse class sections",
		Long:  "List, view, and manage class sections including schedules and calendars.",
		Example: `  # List all sections
  life-ustc section

  # Search sections by keyword
  life-ustc section -s "calculus"

  # View a specific section
  life-ustc section view <jw-id>`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSectionList(cmd, opts)
		},
	}
	addSectionListFlags(cmd, &opts)
	cmd.AddCommand(newCmdList())
	cmd.AddCommand(newCmdView())
	cmd.AddCommand(newCmdSchedules())
	cmd.AddCommand(newCmdCalendar())
	cmd.AddCommand(newCmdMatchCodes())
	cmd.AddCommand(homework.NewCmdSectionHomework())
	cmd.AddCommand(comment.NewCmdCommentFor("section"))
	cmd.AddCommand(description.NewCmdDescriptionFor("section"))
	return cmd
}

type sectionListOpts struct {
	courseID      string
	semesterID   string
	campusID     string
	departmentID string
	teacherID    string
	search       string
	ids          string
	page         int
	limit        int
}

func runSectionList(cmd *cobra.Command, opts sectionListOpts) error {
	c, err := api.NewTypedClient(cmdutil.ServerFromCmd(cmd), false)
	if err != nil {
		return err
	}
	params := openapi.ListSectionsParams{}
	if opts.courseID != "" {
		params.CourseId = &opts.courseID
	}
	if opts.semesterID != "" {
		params.SemesterId = &opts.semesterID
	}
	if opts.campusID != "" {
		params.CampusId = &opts.campusID
	}
	if opts.departmentID != "" {
		params.DepartmentId = &opts.departmentID
	}
	if opts.teacherID != "" {
		params.TeacherId = &opts.teacherID
	}
	if opts.search != "" {
		params.Search = &opts.search
	}
	if opts.ids != "" {
		params.Ids = &opts.ids
	}
	if opts.page > 0 {
		p := cmdutil.Itoa(opts.page)
		params.Page = &p
	}
	if opts.limit > 0 {
		l := cmdutil.Itoa(opts.limit)
		params.Limit = &l
	}
	data, err := api.ParseResponseRaw(c.ListSections(api.Ctx(), &params))
	if err != nil {
		return err
	}
	raw, rows, total, pg := cmdutil.ExtractList(data)
	output.OutputList(raw, rows, []output.Column{
		{Header: "ID", Key: "id"},
		{Header: "Code", Key: "code"},
		{Header: "Course", Key: "course.namePrimary"},
		{Header: "Semester", Key: "semester.name"},
		{Header: "Campus", Key: "campus.name"},
	}, total, pg)
	return nil
}

func addSectionListFlags(cmd *cobra.Command, opts *sectionListOpts) {
	cmd.Flags().StringVar(&opts.courseID, "course-id", "", "Course ID")
	cmd.Flags().StringVar(&opts.semesterID, "semester-id", "", "Semester ID")
	cmd.Flags().StringVar(&opts.campusID, "campus-id", "", "Campus ID")
	cmd.Flags().StringVar(&opts.departmentID, "department-id", "", "Department ID")
	cmd.Flags().StringVar(&opts.teacherID, "teacher-id", "", "Teacher ID")
	cmd.Flags().StringVarP(&opts.search, "search", "s", "", "Search query")
	cmd.Flags().StringVar(&opts.ids, "ids", "", "Comma-separated section IDs")
	cmdutil.AddListFlags(cmd, &opts.page, &opts.limit)
}

func newCmdList() *cobra.Command {
	var opts sectionListOpts
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List sections",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSectionList(cmd, opts)
		},
	}
	addSectionListFlags(cmd, &opts)
	return cmd
}

func newCmdView() *cobra.Command {
	return &cobra.Command{
		Use:     "view <jw-id>",
		Aliases: []string{"show"},
		Short:   "View section details",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := api.NewTypedClient(cmdutil.ServerFromCmd(cmd), false)
			if err != nil {
				return err
			}
			data, err := api.ParseResponseRaw(c.GetSection(api.Ctx(), args[0]))
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
				{Key: "Course", Value: output.Resolve(m, "course.namePrimary")},
				{Key: "Semester", Value: output.Resolve(m, "semester.name")},
				{Key: "Campus", Value: output.Resolve(m, "campus.name")},
			}, "Section")

			if teachers, ok := m["teachers"].([]any); ok && len(teachers) > 0 {
				fmt.Println()
				output.Bold("  Teachers")
				var rows []map[string]any
				for _, t := range teachers {
					if row, ok := t.(map[string]any); ok {
						rows = append(rows, row)
					}
				}
				output.Table(rows, []output.Column{
					{Header: "ID", Key: "id"},
					{Header: "Name", Key: "namePrimary"},
					{Header: "Name (EN)", Key: "nameSecondary"},
					{Header: "Department", Key: "department.name"},
				})
			}

			if schedules, ok := m["schedules"].([]any); ok && len(schedules) > 0 {
				fmt.Println()
				output.Bold("  Schedules")
				var rows []map[string]any
				for _, s := range schedules {
					if row, ok := s.(map[string]any); ok {
						rows = append(rows, row)
					}
				}
				output.Table(rows, []output.Column{
					{Header: "ID", Key: "id"},
					{Header: "Day", Key: "weekday"},
					{Header: "Start", Key: "startTime"},
					{Header: "End", Key: "endTime"},
					{Header: "Place", Key: "place"},
				})
			}
			return nil
		},
	}
}

func newCmdSchedules() *cobra.Command {
	return &cobra.Command{
		Use:   "schedules <jw-id>",
		Short: "List schedules for a section",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := api.NewTypedClient(cmdutil.ServerFromCmd(cmd), false)
			if err != nil {
				return err
			}
			data, err := api.ParseResponseRaw(c.GetSectionSchedules(api.Ctx(), args[0]))
			if err != nil {
				return err
			}
			_, rows, total, pg := cmdutil.ExtractList(data)
			output.OutputList(data, rows, []output.Column{
				{Header: "ID", Key: "id"},
				{Header: "Day", Key: "weekday"},
				{Header: "Start", Key: "startTime"},
				{Header: "End", Key: "endTime"},
				{Header: "Place", Key: "place"},
			}, total, pg)
			return nil
		},
	}
}

func newCmdCalendar() *cobra.Command {
	var outFile string
	cmd := &cobra.Command{
		Use:   "calendar <jw-id>",
		Short: "Download ICS calendar for a section",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := api.NewTypedClient(cmdutil.ServerFromCmd(cmd), false)
			if err != nil {
				return err
			}
			resp, err := c.GetSectionCalendar(api.Ctx(), args[0])
			if err != nil {
				return err
			}
			defer func() { _ = resp.Body.Close() }()
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return err
			}
			if outFile != "" {
				if err := os.WriteFile(outFile, body, 0o644); err != nil {
					return err
				}
				output.Success(fmt.Sprintf("Saved to %s", outFile))
			} else {
				fmt.Print(string(body))
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&outFile, "output", "o", "", "Save to file")
	return cmd
}

func newCmdMatchCodes() *cobra.Command {
	var semesterID string
	cmd := &cobra.Command{
		Use:   "match-codes <code>...",
		Short: "Match section codes",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := api.NewTypedClient(cmdutil.ServerFromCmd(cmd), false)
			if err != nil {
				return err
			}
			body := openapi.MatchSectionCodesJSONRequestBody{
				Codes: args,
			}
			if semesterID != "" {
				body.SemesterId = &semesterID
			}
			data, err := api.ParseResponseRaw(c.MatchSectionCodes(api.Ctx(), body))
			if err != nil {
				return err
			}
			if output.IsJSON() {
				output.JSON(data)
				return nil
			}
			_, rows, total, pg := cmdutil.ExtractList(data, "sections")
			output.OutputList(data, rows, []output.Column{
				{Header: "ID", Key: "id"},
				{Header: "Code", Key: "code"},
				{Header: "Course", Key: "course.nameCn"},
				{Header: "Semester", Key: "semester.nameCn"},
			}, total, pg)
			m := cmdutil.AsMap(data)
			if unmatched, ok := m["unmatchedCodes"].([]any); ok && len(unmatched) > 0 {
				fmt.Println()
				output.Warning(fmt.Sprintf("%d code(s) not found: %v", len(unmatched), unmatched))
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&semesterID, "semester-id", "", "Semester ID filter")
	return cmd
}

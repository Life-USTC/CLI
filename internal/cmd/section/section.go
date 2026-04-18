package section

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Life-USTC/CLI/internal/api"
	"github.com/Life-USTC/CLI/internal/cmd/cmdutil"
	"github.com/Life-USTC/CLI/internal/output"
)

func NewCmdSection() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "section <command>",
		Short: "Browse class sections",
	}
	cmd.AddCommand(newCmdList())
	cmd.AddCommand(newCmdView())
	cmd.AddCommand(newCmdSchedules())
	cmd.AddCommand(newCmdCalendar())
	cmd.AddCommand(newCmdMatchCodes())
	return cmd
}

func newCmdList() *cobra.Command {
	var (
		courseID, semesterID, campusID, teacherID, search, ids string
		page, limit                                            int
	)
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List sections",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := api.NewClient(cmdutil.ServerFromCmd(cmd), false)
			if err != nil {
				return err
			}
			params := url.Values{}
			if courseID != "" {
				params.Set("courseId", courseID)
			}
			if semesterID != "" {
				params.Set("semesterId", semesterID)
			}
			if campusID != "" {
				params.Set("campusId", campusID)
			}
			if teacherID != "" {
				params.Set("teacherId", teacherID)
			}
			if search != "" {
				params.Set("search", search)
			}
			if ids != "" {
				params.Set("ids", ids)
			}
			if page > 0 {
				params.Set("page", cmdutil.Itoa(page))
			}
			if limit > 0 {
				params.Set("limit", cmdutil.Itoa(limit))
			}
			data, err := c.Get("/api/sections", params)
			if err != nil {
				return err
			}
			raw, rows, total, pg := cmdutil.ExtractList(data)
			output.OutputList(raw, rows, []output.Column{
				{Header: "Code", Key: "code"},
				{Header: "Course", Key: "course.namePrimary"},
				{Header: "Semester", Key: "semester.name"},
				{Header: "Campus", Key: "campus.name"},
			}, total, pg)
			return nil
		},
	}
	cmd.Flags().StringVar(&courseID, "course-id", "", "Course ID")
	cmd.Flags().StringVar(&semesterID, "semester-id", "", "Semester ID")
	cmd.Flags().StringVar(&campusID, "campus-id", "", "Campus ID")
	cmd.Flags().StringVar(&teacherID, "teacher-id", "", "Teacher ID")
	cmd.Flags().StringVarP(&search, "search", "s", "", "Search query")
	cmd.Flags().StringVar(&ids, "ids", "", "Comma-separated section IDs")
	cmd.Flags().IntVar(&page, "page", 0, "Page number")
	cmd.Flags().IntVar(&limit, "limit", 0, "Results per page")
	return cmd
}

func newCmdView() *cobra.Command {
	return &cobra.Command{
		Use:   "view <jw-id>",
		Short: "View section details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := api.NewClient(cmdutil.ServerFromCmd(cmd), false)
			if err != nil {
				return err
			}
			data, err := c.Get(fmt.Sprintf("/api/sections/%s", args[0]), nil)
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
			c, err := api.NewClient(cmdutil.ServerFromCmd(cmd), false)
			if err != nil {
				return err
			}
			data, err := c.Get(fmt.Sprintf("/api/sections/%s/schedules", args[0]), nil)
			if err != nil {
				return err
			}
			_, rows, total, pg := cmdutil.ExtractList(data)
			output.OutputList(data, rows, []output.Column{
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
			c, err := api.NewClient(cmdutil.ServerFromCmd(cmd), false)
			if err != nil {
				return err
			}
			resp, err := c.GetRaw(fmt.Sprintf("/api/sections/%s/calendar.ics", args[0]), nil)
			if err != nil {
				return err
			}
			defer resp.Body.Close()
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
			c, err := api.NewClient(cmdutil.ServerFromCmd(cmd), false)
			if err != nil {
				return err
			}
			body := map[string]any{"codes": strings.Join(args, ",")}
			if semesterID != "" {
				body["semesterId"] = semesterID
			}
			data, err := c.Post("/api/sections/match-codes", body)
			if err != nil {
				return err
			}
			_, rows, total, pg := cmdutil.ExtractList(data)
			output.OutputList(data, rows, []output.Column{
				{Header: "Code", Key: "code"},
				{Header: "Course", Key: "course.namePrimary"},
				{Header: "Semester", Key: "semester.name"},
			}, total, pg)
			return nil
		},
	}
	cmd.Flags().StringVar(&semesterID, "semester-id", "", "Semester ID filter")
	return cmd
}

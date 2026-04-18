package schedule

import (
	"net/url"

	"github.com/spf13/cobra"

	"github.com/Life-USTC/CLI/internal/api"
	"github.com/Life-USTC/CLI/internal/cmd/cmdutil"
	"github.com/Life-USTC/CLI/internal/output"
)

var weekdays = map[string]string{
	"1": "Mon", "2": "Tue", "3": "Wed", "4": "Thu",
	"5": "Fri", "6": "Sat", "7": "Sun",
}

func NewCmdSchedule() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "schedule <command>",
		Short: "Browse schedules",
	}
	cmd.AddCommand(newCmdList())
	return cmd
}

func newCmdList() *cobra.Command {
	var (
		sectionID, teacherID, dateFrom, dateTo string
		weekday                                int
		page, limit                            int
	)
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List schedules",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := api.NewClient(cmdutil.ServerFromCmd(cmd), false)
			if err != nil {
				return err
			}
			params := url.Values{}
			if sectionID != "" {
				params.Set("sectionId", sectionID)
			}
			if teacherID != "" {
				params.Set("teacherId", teacherID)
			}
			if dateFrom != "" {
				params.Set("dateFrom", dateFrom)
			}
			if dateTo != "" {
				params.Set("dateTo", dateTo)
			}
			if weekday > 0 {
				params.Set("weekday", cmdutil.Itoa(weekday))
			}
			if page > 0 {
				params.Set("page", cmdutil.Itoa(page))
			}
			if limit > 0 {
				params.Set("limit", cmdutil.Itoa(limit))
			}
			data, err := c.Get("/api/schedules", params)
			if err != nil {
				return err
			}
			raw, rows, total, pg := cmdutil.ExtractList(data)

			// Translate weekday numbers to names
			for _, row := range rows {
				if wd, ok := row["weekday"]; ok {
					if s, ok := wd.(string); ok {
						if name, ok := weekdays[s]; ok {
							row["weekday"] = name
						}
					}
					if f, ok := wd.(float64); ok {
						if name, ok := weekdays[cmdutil.Itoa(int(f))]; ok {
							row["weekday"] = name
						}
					}
				}
			}

			output.OutputList(raw, rows, []output.Column{
				{Header: "Course", Key: "course.namePrimary"},
				{Header: "Code", Key: "section.code"},
				{Header: "Day", Key: "weekday"},
				{Header: "Start", Key: "startTime"},
				{Header: "End", Key: "endTime"},
				{Header: "Place", Key: "place"},
			}, total, pg)
			return nil
		},
	}
	cmd.Flags().StringVar(&sectionID, "section-id", "", "Section ID")
	cmd.Flags().StringVar(&teacherID, "teacher-id", "", "Teacher ID")
	cmd.Flags().StringVar(&dateFrom, "date-from", "", "Start date")
	cmd.Flags().StringVar(&dateTo, "date-to", "", "End date")
	cmd.Flags().IntVar(&weekday, "weekday", 0, "Weekday (1=Mon, 7=Sun)")
	cmd.Flags().IntVar(&page, "page", 0, "Page number")
	cmd.Flags().IntVar(&limit, "limit", 0, "Results per page")
	return cmd
}

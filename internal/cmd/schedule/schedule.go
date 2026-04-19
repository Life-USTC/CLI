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

type scheduleListOpts struct {
	sectionID, teacherID, dateFrom, dateTo string
	weekday                                int
	page, limit                            int
}

func NewCmdSchedule() *cobra.Command {
	opts := scheduleListOpts{}
	cmd := &cobra.Command{
		Use:   "schedule [command]",
		Short: "Browse schedules",
		Long:  "List class schedules with optional filters for section, teacher, date range, and weekday.",
		Example: `  # List all schedules
  life-ustc schedule

  # Filter by weekday
  life-ustc schedule list --weekday 3`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runScheduleList(cmd, opts)
		},
	}
	addScheduleListFlags(cmd, &opts)
	cmd.AddCommand(newCmdList())
	return cmd
}

func addScheduleListFlags(cmd *cobra.Command, opts *scheduleListOpts) {
	cmd.Flags().StringVar(&opts.sectionID, "section-id", "", "Section ID")
	cmd.Flags().StringVar(&opts.teacherID, "teacher-id", "", "Teacher ID")
	cmd.Flags().StringVar(&opts.dateFrom, "date-from", "", "Start date")
	cmd.Flags().StringVar(&opts.dateTo, "date-to", "", "End date")
	cmd.Flags().IntVar(&opts.weekday, "weekday", 0, "Weekday (1=Mon, 7=Sun)")
	cmdutil.AddListFlags(cmd, &opts.page, &opts.limit)
}

func runScheduleList(cmd *cobra.Command, opts scheduleListOpts) error {
	c, err := api.NewClient(cmdutil.ServerFromCmd(cmd), false)
	if err != nil {
		return err
	}
	params := url.Values{}
	if opts.sectionID != "" {
		params.Set("sectionId", opts.sectionID)
	}
	if opts.teacherID != "" {
		params.Set("teacherId", opts.teacherID)
	}
	if opts.dateFrom != "" {
		params.Set("dateFrom", opts.dateFrom)
	}
	if opts.dateTo != "" {
		params.Set("dateTo", opts.dateTo)
	}
	if opts.weekday > 0 {
		params.Set("weekday", cmdutil.Itoa(opts.weekday))
	}
	cmdutil.ApplyListParams(params, opts.page, opts.limit)
	data, err := c.Get("/api/schedules", params)
	if err != nil {
		return err
	}
	raw, rows, total, pg := cmdutil.ExtractList(data)

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
		{Header: "ID", Key: "id"},
		{Header: "Course", Key: "course.namePrimary"},
		{Header: "Code", Key: "section.code"},
		{Header: "Day", Key: "weekday"},
		{Header: "Start", Key: "startTime"},
		{Header: "End", Key: "endTime"},
		{Header: "Place", Key: "place"},
	}, total, pg)
	return nil
}

func newCmdList() *cobra.Command {
	opts := scheduleListOpts{}
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List schedules",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runScheduleList(cmd, opts)
		},
	}
	addScheduleListFlags(cmd, &opts)
	return cmd
}

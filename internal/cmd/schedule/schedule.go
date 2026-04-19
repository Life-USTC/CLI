package schedule

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/Life-USTC/CLI/internal/api"
	"github.com/Life-USTC/CLI/internal/cmd/cmdutil"
	openapi "github.com/Life-USTC/CLI/internal/openapi"
	"github.com/Life-USTC/CLI/internal/output"
)

var weekdays = map[string]string{
	"1": "Mon", "2": "Tue", "3": "Wed", "4": "Thu",
	"5": "Fri", "6": "Sat", "7": "Sun",
}

type scheduleListOpts struct {
	sectionID, teacherID, roomID, dateFrom, dateTo string
	weekday                                        int
	page, limit                                    int
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
	cmd.Flags().StringVar(&opts.roomID, "room-id", "", "Room ID")
	cmd.Flags().StringVar(&opts.dateFrom, "date-from", "", "Start date")
	cmd.Flags().StringVar(&opts.dateTo, "date-to", "", "End date")
	cmd.Flags().IntVar(&opts.weekday, "weekday", 0, "Weekday (1=Mon, 7=Sun)")
	cmdutil.AddListFlags(cmd, &opts.page, &opts.limit)
}

func runScheduleList(cmd *cobra.Command, opts scheduleListOpts) error {
	c, err := api.NewTypedClient(cmdutil.ServerFromCmd(cmd), false)
	if err != nil {
		return err
	}
	params := openapi.ListSchedulesParams{}
	if opts.sectionID != "" {
		params.SectionId = &opts.sectionID
	}
	if opts.teacherID != "" {
		params.TeacherId = &opts.teacherID
	}
	if opts.roomID != "" {
		params.RoomId = &opts.roomID
	}
	if opts.dateFrom != "" {
		t, err := time.Parse(time.DateOnly, opts.dateFrom)
		if err != nil {
			return fmt.Errorf("invalid --date-from: %w", err)
		}
		params.DateFrom = &t
	}
	if opts.dateTo != "" {
		t, err := time.Parse(time.DateOnly, opts.dateTo)
		if err != nil {
			return fmt.Errorf("invalid --date-to: %w", err)
		}
		params.DateTo = &t
	}
	if opts.weekday > 0 {
		w := cmdutil.Itoa(opts.weekday)
		params.Weekday = &w
	}
	if opts.page > 0 {
		p := cmdutil.Itoa(opts.page)
		params.Page = &p
	}
	if opts.limit > 0 {
		l := cmdutil.Itoa(opts.limit)
		params.Limit = &l
	}
	data, err := api.ParseResponseRaw(c.ListSchedules(api.Ctx(), &params))
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

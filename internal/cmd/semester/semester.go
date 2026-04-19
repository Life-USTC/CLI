package semester

import (
	"net/url"

	"github.com/spf13/cobra"

	"github.com/Life-USTC/CLI/internal/api"
	"github.com/Life-USTC/CLI/internal/cmd/cmdutil"
	"github.com/Life-USTC/CLI/internal/output"
)

type semesterListOpts struct {
	page, limit int
}

func NewCmdSemester() *cobra.Command {
	opts := semesterListOpts{}
	cmd := &cobra.Command{
		Use:   "semester [command]",
		Short: "Browse semesters",
		Long:  "List and inspect academic semesters.",
		Example: `  # List all semesters
  life-ustc semester

  # Show the current semester
  life-ustc semester current`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSemesterList(cmd, opts)
		},
	}
	cmdutil.AddListFlags(cmd, &opts.page, &opts.limit)
	cmd.AddCommand(newCmdList())
	cmd.AddCommand(newCmdCurrent())
	return cmd
}

func runSemesterList(cmd *cobra.Command, opts semesterListOpts) error {
	c, err := api.NewClient(cmdutil.ServerFromCmd(cmd), false)
	if err != nil {
		return err
	}
	params := url.Values{}
	cmdutil.ApplyListParams(params, opts.page, opts.limit)
	data, err := c.Get("/api/semesters", params)
	if err != nil {
		return err
	}
	raw, rows, total, pg := cmdutil.ExtractList(data)
	output.OutputList(raw, rows, []output.Column{
		{Header: "ID", Key: "id"},
		{Header: "Code", Key: "code"},
		{Header: "Name", Key: "nameCn"},
		{Header: "Start", Key: "startDate"},
		{Header: "End", Key: "endDate"},
	}, total, pg)
	return nil
}

func newCmdList() *cobra.Command {
	opts := semesterListOpts{}
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List semesters",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSemesterList(cmd, opts)
		},
	}
	cmdutil.AddListFlags(cmd, &opts.page, &opts.limit)
	return cmd
}

func newCmdCurrent() *cobra.Command {
	return &cobra.Command{
		Use:   "current",
		Short: "Show current semester",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := api.NewClient(cmdutil.ServerFromCmd(cmd), false)
			if err != nil {
				return err
			}
			data, err := c.Get("/api/semesters/current", nil)
			if err != nil {
				return err
			}
			output.OutputDetail(data, []output.FieldDef{
				{Key: "id", Label: "ID"},
				{Key: "code", Label: "Code"},
				{Key: "nameCn", Label: "Name"},
				{Key: "startDate", Label: "Start"},
				{Key: "endDate", Label: "End"},
			}, "Current semester")
			return nil
		},
	}
}

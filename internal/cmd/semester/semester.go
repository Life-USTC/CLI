package semester

import (
	"github.com/spf13/cobra"

	"github.com/Life-USTC/CLI/internal/api"
	"github.com/Life-USTC/CLI/internal/cmd/cmdutil"
	"github.com/Life-USTC/CLI/internal/output"
)

func NewCmdSemester() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "semester <command>",
		Short: "Browse semesters",
	}
	cmd.AddCommand(newCmdList())
	cmd.AddCommand(newCmdCurrent())
	return cmd
}

func newCmdList() *cobra.Command {
	var page, limit int
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List semesters",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := api.NewClient(cmdutil.ServerFromCmd(cmd), false)
			if err != nil {
				return err
			}
			params := make(map[string][]string)
			if page > 0 {
				params["page"] = []string{cmdutil.Itoa(page)}
			}
			if limit > 0 {
				params["limit"] = []string{cmdutil.Itoa(limit)}
			}
			data, err := c.Get("/api/semesters", params)
			if err != nil {
				return err
			}
			_, rows, total, pg := cmdutil.ExtractList(data)
			output.OutputList(data, rows, []output.Column{
				{Header: "Code", Key: "code"},
				{Header: "Name", Key: "nameCn"},
				{Header: "Start", Key: "startDate"},
				{Header: "End", Key: "endDate"},
			}, total, pg)
			return nil
		},
	}
	cmd.Flags().IntVar(&page, "page", 0, "Page number")
	cmd.Flags().IntVar(&limit, "limit", 0, "Results per page")
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
				{Key: "code", Label: "Code"},
				{Key: "nameCn", Label: "Name"},
				{Key: "startDate", Label: "Start"},
				{Key: "endDate", Label: "End"},
			}, "Current semester")
			return nil
		},
	}
}

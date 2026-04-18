package metadata

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Life-USTC/CLI/internal/api"
	"github.com/Life-USTC/CLI/internal/cmd/cmdutil"
	"github.com/Life-USTC/CLI/internal/output"
)

type category struct {
	key  string
	name string
	cols []output.Column
}

var categories = []category{
	{"educationLevels", "Education Levels", []output.Column{{Header: "Name", Key: "nameCn"}, {Header: "ID", Key: "id"}}},
	{"campuses", "Campuses", []output.Column{{Header: "Name", Key: "nameCn"}, {Header: "ID", Key: "id"}}},
	{"courseCategories", "Categories", []output.Column{{Header: "Name", Key: "nameCn"}, {Header: "ID", Key: "id"}}},
	{"classTypes", "Class Types", []output.Column{{Header: "Name", Key: "nameCn"}, {Header: "ID", Key: "id"}}},
	{"courseGradations", "Gradations", []output.Column{{Header: "Name", Key: "nameCn"}, {Header: "ID", Key: "id"}}},
	{"courseTypes", "Course Types", []output.Column{{Header: "Name", Key: "nameCn"}, {Header: "ID", Key: "id"}}},
	{"examModes", "Exam Modes", []output.Column{{Header: "Name", Key: "nameCn"}, {Header: "ID", Key: "id"}}},
	{"teachLanguages", "Teaching Languages", []output.Column{{Header: "Name", Key: "nameCn"}, {Header: "ID", Key: "id"}}},
	{"courseClassifies", "Classifies", []output.Column{{Header: "Name", Key: "nameCn"}, {Header: "ID", Key: "id"}}},
}

func NewCmdMetadata() *cobra.Command {
	return &cobra.Command{
		Use:   "metadata",
		Short: "Show platform metadata dictionaries (campuses, categories, ...)",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := api.NewClient(cmdutil.ServerFromCmd(cmd), false)
			if err != nil {
				return err
			}
			data, err := c.Get("/api/metadata", nil)
			if err != nil {
				return err
			}
			if output.IsJSON() {
				output.JSON(data)
				return nil
			}

			m, ok := data.(map[string]any)
			if !ok {
				output.JSON(data)
				return nil
			}

			for _, cat := range categories {
				items, ok := m[cat.key].([]any)
				if !ok || len(items) == 0 {
					continue
				}
				fmt.Println()
				output.Bold(fmt.Sprintf("  %s  (%d)", cat.name, len(items)))
				var rows []map[string]any
				for _, item := range items {
					if row, ok := item.(map[string]any); ok {
						rows = append(rows, row)
					}
				}
				output.Table(rows, cat.cols)
			}
			return nil
		},
	}
}

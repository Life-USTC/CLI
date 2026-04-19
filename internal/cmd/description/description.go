package description

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Life-USTC/CLI/internal/api"
	"github.com/Life-USTC/CLI/internal/cmd/cmdutil"
	"github.com/Life-USTC/CLI/internal/output"
)

func NewCmdDescription() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "description <command>",
		Short: "View and edit resource descriptions",
	}
	cmd.AddCommand(newCmdGet())
	cmd.AddCommand(newCmdSet())
	return cmd
}

// NewCmdDescriptionFor creates a "description" command tree scoped to a target type.
// get and set take the target ID as a positional argument.
func NewCmdDescriptionFor(targetType string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "description <command>",
		Short: fmt.Sprintf("View and edit %s descriptions", targetType),
	}
	cmd.AddCommand(newCmdGetFor(targetType))
	cmd.AddCommand(newCmdSetFor(targetType))
	return cmd
}

func newCmdGetFor(targetType string) *cobra.Command {
	return &cobra.Command{
		Use:     fmt.Sprintf("get <%s-id>", targetType),
		Aliases: []string{"show"},
		Short:   fmt.Sprintf("Get description for a %s", targetType),
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := api.NewClient(cmdutil.ServerFromCmd(cmd), false)
			if err != nil {
				return err
			}
			data, err := c.Get("/api/descriptions", map[string][]string{
				"targetType": {targetType},
				"targetId":   {args[0]},
			})
			if err != nil {
				return err
			}
			if output.IsJSON() {
				output.JSON(data)
				return nil
			}
			m := cmdutil.AsMap(data)
			content := ""
			if c, ok := m["content"].(string); ok {
				content = c
			} else if c, ok := m["description"].(string); ok {
				content = c
			}
			if content != "" {
				fmt.Println()
				fmt.Println(content)
			} else {
				output.Dim("  No description.")
			}

			if history, ok := m["history"].([]any); ok && len(history) > 0 {
				fmt.Println()
				output.Bold("  History")
				var rows []map[string]any
				for _, h := range history {
					if row, ok := h.(map[string]any); ok {
						rows = append(rows, row)
					}
				}
				output.Table(rows, []output.Column{
					{Header: "Updated", Key: "updatedAt"},
					{Header: "By", Key: "updatedBy.name"},
				})
			}
			return nil
		},
	}
}

func newCmdSetFor(targetType string) *cobra.Command {
	var content string
	cmd := &cobra.Command{
		Use:   fmt.Sprintf("set <%s-id>", targetType),
		Short: fmt.Sprintf("Create or update description for a %s", targetType),
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := api.NewClient(cmdutil.ServerFromCmd(cmd), true)
			if err != nil {
				return err
			}
			data, err := c.Post("/api/descriptions", map[string]any{
				"targetType": targetType,
				"targetId":   args[0],
				"content":    content,
			})
			if err != nil {
				return err
			}
			m := cmdutil.AsMap(data)
			if updated, _ := m["updated"].(bool); updated {
				output.Success("Description updated.")
			} else {
				output.Success("Description created.")
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&content, "content", "c", "", "Description content (Markdown)")
	_ = cmd.MarkFlagRequired("content")
	return cmd
}

func newCmdGet() *cobra.Command {
	var targetType, targetID string
	cmd := &cobra.Command{
		Use:     "get",
		Aliases: []string{"show"},
		Short:   "Get description for a resource",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := api.NewClient(cmdutil.ServerFromCmd(cmd), false)
			if err != nil {
				return err
			}
			data, err := c.Get("/api/descriptions", map[string][]string{
				"targetType": {targetType},
				"targetId":   {targetID},
			})
			if err != nil {
				return err
			}
			if output.IsJSON() {
				output.JSON(data)
				return nil
			}
			m := cmdutil.AsMap(data)
			content := ""
			if c, ok := m["content"].(string); ok {
				content = c
			} else if c, ok := m["description"].(string); ok {
				content = c
			}
			if content != "" {
				fmt.Println()
				fmt.Println(content)
			} else {
				output.Dim("  No description.")
			}

			if history, ok := m["history"].([]any); ok && len(history) > 0 {
				fmt.Println()
				output.Bold("  History")
				var rows []map[string]any
				for _, h := range history {
					if row, ok := h.(map[string]any); ok {
						rows = append(rows, row)
					}
				}
				output.Table(rows, []output.Column{
					{Header: "ID", Key: "id"},
					{Header: "Updated", Key: "updatedAt"},
					{Header: "By", Key: "updatedBy.name"},
				})
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&targetType, "target-type", "", "Target type (section, course, teacher, homework)")
	cmd.Flags().StringVar(&targetID, "target-id", "", "Target ID")
	_ = cmd.MarkFlagRequired("target-type")
	_ = cmd.MarkFlagRequired("target-id")
	return cmd
}

func newCmdSet() *cobra.Command {
	var targetType, targetID, content string
	cmd := &cobra.Command{
		Use:   "set",
		Short: "Create or update a description",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := api.NewClient(cmdutil.ServerFromCmd(cmd), true)
			if err != nil {
				return err
			}
			data, err := c.Post("/api/descriptions", map[string]any{
				"targetType": targetType,
				"targetId":   targetID,
				"content":    content,
			})
			if err != nil {
				return err
			}
			m := cmdutil.AsMap(data)
			if updated, _ := m["updated"].(bool); updated {
				output.Success("Description updated.")
			} else {
				output.Success("Description created.")
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&targetType, "target-type", "", "Target type")
	cmd.Flags().StringVar(&targetID, "target-id", "", "Target ID")
	cmd.Flags().StringVarP(&content, "content", "c", "", "Description content (Markdown)")
	_ = cmd.MarkFlagRequired("target-type")
	_ = cmd.MarkFlagRequired("target-id")
	_ = cmd.MarkFlagRequired("content")
	return cmd
}

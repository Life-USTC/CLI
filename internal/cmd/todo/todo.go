package todo

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/Life-USTC/CLI/internal/api"
	"github.com/Life-USTC/CLI/internal/cmd/cmdutil"
	"github.com/Life-USTC/CLI/internal/output"
)

func isInteractive() bool {
	return term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd()))
}

func promptText(label string) string {
	fmt.Printf("%s: ", label)
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		return strings.TrimSpace(scanner.Text())
	}
	return ""
}

func promptSelect(label string, choices []string) string {
	fmt.Printf("%s:\n", label)
	for i, c := range choices {
		fmt.Printf("  %d) %s\n", i+1, c)
	}
	fmt.Print("Choice: ")
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		text := strings.TrimSpace(scanner.Text())
		for i, c := range choices {
			if text == cmdutil.Itoa(i+1) || strings.EqualFold(text, c) {
				return c
			}
		}
		return text
	}
	return choices[0]
}

func NewCmdTodo() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "todo [command]",
		Short: "Manage personal todos",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTodoList(cmd)
		},
	}
	cmd.AddCommand(newCmdList())
	cmd.AddCommand(newCmdCreate())
	cmd.AddCommand(newCmdUpdate())
	cmd.AddCommand(newCmdDelete())
	return cmd
}

func runTodoList(cmd *cobra.Command) error {
	c, err := api.NewClient(cmdutil.ServerFromCmd(cmd), true)
	if err != nil {
		return err
	}
	data, err := c.Get("/api/todos", nil)
	if err != nil {
		return err
	}
	_, rows, total, pg := cmdutil.ExtractList(data)
	output.OutputList(data, rows, []output.Column{
		{Header: "Title", Key: "title"},
		{Header: "Priority", Key: "priority"},
		{Header: "Done", Key: "isCompleted"},
		{Header: "Due", Key: "dueAt"},
	}, total, pg)
	return nil
}

func newCmdList() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List your todos",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTodoList(cmd)
		},
	}
}

func newCmdCreate() *cobra.Command {
	var (
		title, content, priority, due string
	)
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a todo",
		Long:  "Create a todo. When run interactively without --title, prompts for input.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if title == "" {
				if !isInteractive() {
					return fmt.Errorf("--title is required in non-interactive mode")
				}
				title = promptText("Title")
				if title == "" {
					return fmt.Errorf("title is required")
				}
				if content == "" {
					content = promptText("Content (optional)")
				}
				if priority == "" {
					priority = promptSelect("Priority", []string{"low", "medium", "high"})
				}
				if due == "" {
					due = promptText("Due date (optional, ISO 8601)")
				}
			}

			c, err := api.NewClient(cmdutil.ServerFromCmd(cmd), true)
			if err != nil {
				return err
			}
			body := map[string]any{"title": title}
			if content != "" {
				body["content"] = content
			}
			if priority != "" {
				body["priority"] = priority
			}
			if due != "" {
				body["dueAt"] = due
			}
			data, err := c.Post("/api/todos", body)
			if err != nil {
				return err
			}
			m := cmdutil.AsMap(data)
			id, _ := m["id"].(string)
			output.Success(fmt.Sprintf("Created todo %s", id))
			return nil
		},
	}
	cmd.Flags().StringVarP(&title, "title", "t", "", "Todo title")
	cmd.Flags().StringVar(&content, "content", "", "Content")
	cmd.Flags().StringVar(&priority, "priority", "", "Priority (low, medium, high)")
	cmd.Flags().StringVar(&due, "due", "", "Due date (ISO 8601)")
	return cmd
}

func newCmdUpdate() *cobra.Command {
	var (
		title, content, priority, due string
		completed                     bool
		notCompleted                  bool
	)
	cmd := &cobra.Command{
		Use:   "update <todo-id>",
		Short: "Update a todo",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := api.NewClient(cmdutil.ServerFromCmd(cmd), true)
			if err != nil {
				return err
			}
			body := map[string]any{}
			if title != "" {
				body["title"] = title
			}
			if content != "" {
				body["content"] = content
			}
			if priority != "" {
				body["priority"] = priority
			}
			if due != "" {
				body["dueAt"] = due
			}
			if completed {
				body["isCompleted"] = true
			}
			if notCompleted {
				body["isCompleted"] = false
			}
			if len(body) == 0 {
				return fmt.Errorf("nothing to update")
			}
			_, err = c.Patch(fmt.Sprintf("/api/todos/%s", args[0]), body)
			if err != nil {
				return err
			}
			output.Success("Todo updated.")
			return nil
		},
	}
	cmd.Flags().StringVarP(&title, "title", "t", "", "Title")
	cmd.Flags().StringVar(&content, "content", "", "Content")
	cmd.Flags().StringVar(&priority, "priority", "", "Priority")
	cmd.Flags().StringVar(&due, "due", "", "Due date")
	cmd.Flags().BoolVar(&completed, "completed", false, "Mark completed")
	cmd.Flags().BoolVar(&notCompleted, "not-completed", false, "Mark not completed")
	return cmd
}

func newCmdDelete() *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "delete <todo-id>",
		Short: "Delete a todo",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !yes {
				fmt.Print("Delete this todo? (y/N) ")
				scanner := bufio.NewScanner(os.Stdin)
				if scanner.Scan() && strings.ToLower(strings.TrimSpace(scanner.Text())) != "y" {
					fmt.Println("Cancelled.")
					return nil
				}
			}
			c, err := api.NewClient(cmdutil.ServerFromCmd(cmd), true)
			if err != nil {
				return err
			}
			_, err = c.Delete(fmt.Sprintf("/api/todos/%s", args[0]), nil)
			if err != nil {
				return err
			}
			output.Success("Todo deleted.")
			return nil
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip confirmation")
	return cmd
}

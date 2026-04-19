package todo

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/Life-USTC/CLI/internal/api"
	"github.com/Life-USTC/CLI/internal/cmd/cmdutil"
	openapi "github.com/Life-USTC/CLI/internal/openapi"
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
	var opts todoListOpts
	cmd := &cobra.Command{
		Use:   "todo [command]",
		Short: "Manage personal todos",
		Long:  "Create, list, update, and delete personal todo items.",
		Example: `  # List all pending todos
  life-ustc me todo --pending

  # Create a new todo
  life-ustc me todo create --title "Review notes" --priority high --due 2025-06-01

  # Mark a todo as done
  life-ustc me todo update <id> --completed

  # Get todo IDs for scripting
  life-ustc me todo --jq '.[].id'`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTodoList(cmd, opts)
		},
	}
	addTodoListFlags(cmd, &opts)
	cmd.AddCommand(newCmdList())
	cmd.AddCommand(newCmdCreate())
	cmd.AddCommand(newCmdUpdate())
	cmd.AddCommand(newCmdDelete())
	return cmd
}

type todoListOpts struct {
	done     bool
	pending  bool
	priority string
	before   string
	after    string
	sort     string
	page     int
	limit    int
}

func runTodoList(cmd *cobra.Command, opts todoListOpts) error {
	c, err := api.NewTypedClient(cmdutil.ServerFromCmd(cmd), true)
	if err != nil {
		return err
	}

	// Build server-side filter params
	params := &openapi.ListTodosParams{}
	if opts.done {
		v := openapi.ListTodosParamsCompletedTrue
		params.Completed = &v
	}
	if opts.pending {
		v := openapi.ListTodosParamsCompletedFalse
		params.Completed = &v
	}
	if opts.priority != "" {
		v := openapi.ListTodosParamsPriority(opts.priority)
		params.Priority = &v
	}
	if opts.before != "" {
		t, err := time.Parse(time.RFC3339, opts.before)
		if err != nil {
			return fmt.Errorf("invalid --before date (expected RFC3339): %w", err)
		}
		params.DueBefore = &t
	}
	if opts.after != "" {
		t, err := time.Parse(time.RFC3339, opts.after)
		if err != nil {
			return fmt.Errorf("invalid --after date (expected RFC3339): %w", err)
		}
		params.DueAfter = &t
	}

	data, err := api.ParseResponseRaw(c.ListTodos(api.Ctx(), params))
	if err != nil {
		return err
	}
	_, rows, total, pg := cmdutil.ExtractList(data, "todos")

	output.OutputList(data, rows, []output.Column{
		{Header: "ID", Key: "id"},
		{Header: "Title", Key: "title"},
		{Header: "Priority", Key: "priority"},
		{Header: "Done", Key: "completed"},
		{Header: "Due", Key: "dueAt"},
	}, total, pg)
	return nil
}

func addTodoListFlags(cmd *cobra.Command, opts *todoListOpts) {
	cmd.Flags().BoolVar(&opts.done, "done", false, "Show only completed todos")
	cmd.Flags().BoolVar(&opts.pending, "pending", false, "Show only pending todos")
	cmd.Flags().StringVar(&opts.priority, "priority", "", "Filter by priority (low, medium, high)")
	cmd.Flags().StringVar(&opts.before, "before", "", "Show todos due before this date (ISO 8601)")
	cmd.Flags().StringVar(&opts.after, "after", "", "Show todos due after this date (ISO 8601)")
	cmd.Flags().StringVar(&opts.sort, "sort", "", "Sort by field (created, due, priority)")
	cmdutil.AddListFlags(cmd, &opts.page, &opts.limit)
}

func newCmdList() *cobra.Command {
	var opts todoListOpts
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List your todos",
		Example: `  life-ustc me todo list --pending --priority high
  life-ustc me todo list --done --sort due
  life-ustc me todo list --before 2025-06-01 --limit 10`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTodoList(cmd, opts)
		},
	}
	addTodoListFlags(cmd, &opts)
	return cmd
}

func newCmdCreate() *cobra.Command {
	var (
		title, content, priority, due string
	)
	cmd := &cobra.Command{
		Use:     "create",
		Aliases: []string{"new"},
		Short:   "Create a todo",
		Long:    "Create a todo. When run interactively without --title, prompts for input.",
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

			c, err := api.NewTypedClient(cmdutil.ServerFromCmd(cmd), true)
			if err != nil {
				return err
			}
			body := openapi.CreateTodoJSONRequestBody{Title: title}
			if content != "" {
				body.Content = &content
			}
			if priority != "" {
				p := openapi.TodoCreateRequestSchemaPriority(priority)
				body.Priority = &p
			}
			if due != "" {
				body.DueAt = &due
			}
			data, err := api.ParseResponseRaw(c.CreateTodo(api.Ctx(), nil, body))
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
			c, err := api.NewTypedClient(cmdutil.ServerFromCmd(cmd), true)
			if err != nil {
				return err
			}
			body := openapi.UpdateTodoJSONRequestBody{}
			hasUpdate := false
			if title != "" {
				body.Title = &title
				hasUpdate = true
			}
			if content != "" {
				body.Content = &content
				hasUpdate = true
			}
			if priority != "" {
				p := openapi.TodoUpdateRequestSchemaPriority(priority)
				body.Priority = &p
				hasUpdate = true
			}
			if due != "" {
				body.DueAt = &due
				hasUpdate = true
			}
			if completed {
				t := true
				body.Completed = &t
				hasUpdate = true
			}
			if notCompleted {
				f := false
				body.Completed = &f
				hasUpdate = true
			}
			if !hasUpdate {
				return fmt.Errorf("nothing to update — specify at least one flag (e.g. --title, --completed)")
			}
			_, err = api.ParseResponseRaw(c.UpdateTodo(api.Ctx(), args[0], body))
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
		Use:     "delete <todo-id>",
		Aliases: []string{"rm"},
		Short:   "Delete a todo",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !yes {
				fmt.Print("Delete this todo? (y/N) ")
				scanner := bufio.NewScanner(os.Stdin)
				if scanner.Scan() && strings.ToLower(strings.TrimSpace(scanner.Text())) != "y" {
					output.Warning("Cancelled.")
					return nil
				}
			}
			c, err := api.NewTypedClient(cmdutil.ServerFromCmd(cmd), true)
			if err != nil {
				return err
			}
			_, err = api.ParseResponseRaw(c.DeleteTodo(api.Ctx(), args[0], openapi.DeleteTodoJSONRequestBody{}))
			if err != nil {
				return err
			}
			output.Success("Todo deleted.")
			return nil
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation")
	return cmd
}

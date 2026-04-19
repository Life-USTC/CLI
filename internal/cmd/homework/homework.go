package homework

import (
	"bufio"
	"fmt"
	"net/url"
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
	s := bufio.NewScanner(os.Stdin)
	if s.Scan() {
		return strings.TrimSpace(s.Text())
	}
	return ""
}

func NewCmdHomework() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "homework <command>",
		Short: "Manage section homeworks",
	}
	cmd.AddCommand(newCmdList())
	cmd.AddCommand(newCmdCreate())
	cmd.AddCommand(newCmdUpdate())
	cmd.AddCommand(newCmdDelete())
	cmd.AddCommand(newCmdComplete())
	return cmd
}

// NewCmdSectionHomework returns homework commands scoped to a section.
// list and create take section-id as a positional argument.
func NewCmdSectionHomework() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "homework <command>",
		Short: "Manage section homeworks",
	}
	cmd.AddCommand(newCmdSectionList())
	cmd.AddCommand(newCmdSectionCreate())
	cmd.AddCommand(newCmdUpdate())
	cmd.AddCommand(newCmdDelete())
	return cmd
}

func newCmdSectionList() *cobra.Command {
	var includeDeleted bool
	cmd := &cobra.Command{
		Use:   "list <section-id>",
		Short: "List homeworks for a section",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := api.NewClient(cmdutil.ServerFromCmd(cmd), false)
			if err != nil {
				return err
			}
			params := url.Values{"sectionId": {args[0]}}
			if includeDeleted {
				params.Set("includeDeleted", "true")
			}
			data, err := c.Get("/api/homeworks", params)
			if err != nil {
				return err
			}
			_, rows, total, pg := cmdutil.ExtractList(data)
			output.OutputList(data, rows, []output.Column{
				{Header: "Title", Key: "title"},
				{Header: "Due", Key: "submissionDueAt"},
				{Header: "Major", Key: "isMajor"},
			}, total, pg)
			return nil
		},
	}
	cmd.Flags().BoolVar(&includeDeleted, "include-deleted", false, "Include deleted")
	return cmd
}

func newCmdSectionCreate() *cobra.Command {
	var (
		title, desc, publishedAt, submissionStart, submissionDue string
		major                                                    bool
	)
	cmd := &cobra.Command{
		Use:   "create <section-id>",
		Short: "Create a homework for a section",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sectionID := args[0]
			if title == "" {
				if !isInteractive() {
					return fmt.Errorf("--title is required in non-interactive mode")
				}
				title = promptText("Homework title")
				if desc == "" {
					desc = promptText("Description (optional)")
				}
				if submissionDue == "" {
					submissionDue = promptText("Submission due (optional, ISO 8601)")
				}
			}
			c, err := api.NewClient(cmdutil.ServerFromCmd(cmd), true)
			if err != nil {
				return err
			}
			body := map[string]any{"sectionId": sectionID, "title": title}
			if desc != "" {
				body["description"] = desc
			}
			if publishedAt != "" {
				body["publishedAt"] = publishedAt
			}
			if submissionStart != "" {
				body["submissionStartAt"] = submissionStart
			}
			if submissionDue != "" {
				body["submissionDueAt"] = submissionDue
			}
			if major {
				body["isMajor"] = true
			}
			data, err := c.Post("/api/homeworks", body)
			if err != nil {
				return err
			}
			m := cmdutil.AsMap(data)
			id, _ := m["id"].(string)
			output.Success(fmt.Sprintf("Created homework %s", id))
			return nil
		},
	}
	cmd.Flags().StringVarP(&title, "title", "t", "", "Title")
	cmd.Flags().StringVar(&desc, "description", "", "Description")
	cmd.Flags().StringVar(&publishedAt, "published-at", "", "Publish date (ISO 8601)")
	cmd.Flags().StringVar(&submissionStart, "submission-start", "", "Submission start date")
	cmd.Flags().StringVar(&submissionDue, "submission-due", "", "Submission due date")
	cmd.Flags().BoolVar(&major, "major", false, "Major assignment")
	return cmd
}

// NewCmdMyHomework returns personal homework commands (list + complete).
// Running without a subcommand lists your homeworks.
func NewCmdMyHomework() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "homework [command]",
		Short: "View and manage your homeworks",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMyHomeworkList(cmd)
		},
	}
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List your homeworks",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMyHomeworkList(cmd)
		},
	}
	cmd.AddCommand(listCmd)
	cmd.AddCommand(newCmdComplete())
	return cmd
}

func runMyHomeworkList(cmd *cobra.Command) error {
	c, err := api.NewClient(cmdutil.ServerFromCmd(cmd), true)
	if err != nil {
		return err
	}
	data, err := c.Get("/api/homeworks", nil)
	if err != nil {
		return err
	}
	_, rows, total, pg := cmdutil.ExtractList(data)
	output.OutputList(data, rows, []output.Column{
		{Header: "Title", Key: "title"},
		{Header: "Section", Key: "section.code"},
		{Header: "Due", Key: "submissionDueAt"},
		{Header: "Major", Key: "isMajor"},
		{Header: "Done", Key: "isCompleted"},
	}, total, pg)
	return nil
}

func newCmdList() *cobra.Command {
	var (
		sectionID      string
		includeDeleted bool
	)
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List homeworks for a section",
		RunE: func(cmd *cobra.Command, args []string) error {
			if sectionID == "" {
				return fmt.Errorf("--section-id is required")
			}
			c, err := api.NewClient(cmdutil.ServerFromCmd(cmd), false)
			if err != nil {
				return err
			}
			params := url.Values{"sectionId": {sectionID}}
			if includeDeleted {
				params.Set("includeDeleted", "true")
			}
			data, err := c.Get("/api/homeworks", params)
			if err != nil {
				return err
			}
			_, rows, total, pg := cmdutil.ExtractList(data)
			output.OutputList(data, rows, []output.Column{
				{Header: "Title", Key: "title"},
				{Header: "Due", Key: "submissionDueAt"},
				{Header: "Major", Key: "isMajor"},
			}, total, pg)
			return nil
		},
	}
	cmd.Flags().StringVar(&sectionID, "section-id", "", "Section ID (required)")
	cmd.Flags().BoolVar(&includeDeleted, "include-deleted", false, "Include deleted")
	return cmd
}

func newCmdCreate() *cobra.Command {
	var (
		sectionID, title, desc, publishedAt, submissionStart, submissionDue string
		major                                                               bool
	)
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a homework",
		Long:  "Create a homework. Prompts interactively when --section-id/--title are omitted.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if sectionID == "" || title == "" {
				if !isInteractive() {
					return fmt.Errorf("--section-id and --title are required in non-interactive mode")
				}
				if sectionID == "" {
					sectionID = promptText("Section ID")
				}
				if title == "" {
					title = promptText("Homework title")
				}
				if desc == "" {
					desc = promptText("Description (optional)")
				}
				if submissionDue == "" {
					submissionDue = promptText("Submission due (optional, ISO 8601)")
				}
			}
			c, err := api.NewClient(cmdutil.ServerFromCmd(cmd), true)
			if err != nil {
				return err
			}
			body := map[string]any{"sectionId": sectionID, "title": title}
			if desc != "" {
				body["description"] = desc
			}
			if publishedAt != "" {
				body["publishedAt"] = publishedAt
			}
			if submissionStart != "" {
				body["submissionStartAt"] = submissionStart
			}
			if submissionDue != "" {
				body["submissionDueAt"] = submissionDue
			}
			if major {
				body["isMajor"] = true
			}
			data, err := c.Post("/api/homeworks", body)
			if err != nil {
				return err
			}
			m := cmdutil.AsMap(data)
			id, _ := m["id"].(string)
			output.Success(fmt.Sprintf("Created homework %s", id))
			return nil
		},
	}
	cmd.Flags().StringVar(&sectionID, "section-id", "", "Section ID")
	cmd.Flags().StringVarP(&title, "title", "t", "", "Title")
	cmd.Flags().StringVar(&desc, "description", "", "Description")
	cmd.Flags().StringVar(&publishedAt, "published-at", "", "Publish date (ISO 8601)")
	cmd.Flags().StringVar(&submissionStart, "submission-start", "", "Submission start date")
	cmd.Flags().StringVar(&submissionDue, "submission-due", "", "Submission due date")
	cmd.Flags().BoolVar(&major, "major", false, "Major assignment")
	return cmd
}

func newCmdUpdate() *cobra.Command {
	var (
		title, publishedAt, submissionStart, submissionDue string
		major, notMajor                                    bool
	)
	cmd := &cobra.Command{
		Use:   "update <homework-id>",
		Short: "Update a homework",
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
			if publishedAt != "" {
				body["publishedAt"] = publishedAt
			}
			if submissionStart != "" {
				body["submissionStartAt"] = submissionStart
			}
			if submissionDue != "" {
				body["submissionDueAt"] = submissionDue
			}
			if major {
				body["isMajor"] = true
			}
			if notMajor {
				body["isMajor"] = false
			}
			if len(body) == 0 {
				return fmt.Errorf("nothing to update")
			}
			_, err = c.Patch(fmt.Sprintf("/api/homeworks/%s", args[0]), body)
			if err != nil {
				return err
			}
			output.Success("Homework updated.")
			return nil
		},
	}
	cmd.Flags().StringVarP(&title, "title", "t", "", "Title")
	cmd.Flags().BoolVar(&major, "major", false, "Mark as major")
	cmd.Flags().BoolVar(&notMajor, "not-major", false, "Mark as not major")
	cmd.Flags().StringVar(&publishedAt, "published-at", "", "Publish date")
	cmd.Flags().StringVar(&submissionStart, "submission-start", "", "Submission start")
	cmd.Flags().StringVar(&submissionDue, "submission-due", "", "Submission due")
	return cmd
}

func newCmdDelete() *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "delete <homework-id>",
		Short: "Delete a homework (soft delete)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !yes {
				fmt.Print("Delete this homework? (y/N) ")
				s := bufio.NewScanner(os.Stdin)
				if s.Scan() && strings.ToLower(strings.TrimSpace(s.Text())) != "y" {
					fmt.Println("Cancelled.")
					return nil
				}
			}
			c, err := api.NewClient(cmdutil.ServerFromCmd(cmd), true)
			if err != nil {
				return err
			}
			_, err = c.Delete(fmt.Sprintf("/api/homeworks/%s", args[0]), nil)
			if err != nil {
				return err
			}
			output.Success("Homework deleted.")
			return nil
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip confirmation")
	return cmd
}

func newCmdComplete() *cobra.Command {
	var undo bool
	cmd := &cobra.Command{
		Use:   "complete <homework-id>",
		Short: "Mark homework as complete",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := api.NewClient(cmdutil.ServerFromCmd(cmd), true)
			if err != nil {
				return err
			}
			body := map[string]any{"isCompleted": !undo}
			_, err = c.Put(fmt.Sprintf("/api/homeworks/%s/completion", args[0]), body)
			if err != nil {
				return err
			}
			if undo {
				output.Success("Homework marked as not completed.")
			} else {
				output.Success("Homework marked as completed.")
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&undo, "undo", false, "Mark as not completed")
	return cmd
}

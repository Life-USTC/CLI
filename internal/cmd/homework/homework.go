package homework

import (
	"bufio"
	"fmt"
	"os"
	"strings"

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
		Long:  "List, create, update, and delete homeworks for a course section.",
		Example: `  # List homeworks for a section
  life-ustc section homework list <section-id>

  # Create a homework
  life-ustc section homework create <section-id> --title "Problem Set 1"

  # Delete a homework
  life-ustc section homework delete <homework-id> -y`,
	}
	cmd.AddCommand(newCmdSectionList())
	cmd.AddCommand(newCmdSectionCreate())
	cmd.AddCommand(newCmdUpdate())
	cmd.AddCommand(newCmdDelete())
	return cmd
}

func newCmdSectionList() *cobra.Command {
	var (
		includeDeleted bool
		page, limit    int
	)
	cmd := &cobra.Command{
		Use:     "list <section-id>",
		Aliases: []string{"ls"},
		Short:   "List homeworks for a section",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := api.NewTypedClient(cmdutil.ServerFromCmd(cmd), false)
			if err != nil {
				return err
			}
			inclDel := openapi.ListHomeworksParamsIncludeDeleted("false")
			if includeDeleted {
				inclDel = openapi.ListHomeworksParamsIncludeDeletedTrue
			}
			params := &openapi.ListHomeworksParams{
				SectionId:      &args[0],
				IncludeDeleted: &inclDel,
			}
			data, err := api.ParseResponseRaw(c.ListHomeworks(api.Ctx(), params))
			if err != nil {
				return err
			}
			_, rows, total, pg := cmdutil.ExtractList(data, "homeworks")
			output.OutputList(data, rows, []output.Column{
				{Header: "ID", Key: "id"},
				{Header: "Title", Key: "title"},
				{Header: "Due", Key: "submissionDueAt"},
				{Header: "Major", Key: "isMajor"},
			}, total, pg)
			return nil
		},
	}
	cmd.Flags().BoolVar(&includeDeleted, "include-deleted", false, "Include deleted")
	cmdutil.AddListFlags(cmd, &page, &limit)
	return cmd
}

func newCmdSectionCreate() *cobra.Command {
	var (
		title, desc, publishedAt, submissionStart, submissionDue string
		major, requiresTeam                                      bool
	)
	cmd := &cobra.Command{
		Use:     "create <section-id>",
		Aliases: []string{"new"},
		Short:   "Create a homework for a section",
		Args:    cobra.ExactArgs(1),
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
			c, err := api.NewTypedClient(cmdutil.ServerFromCmd(cmd), true)
			if err != nil {
				return err
			}
			body := openapi.CreateHomeworkJSONRequestBody{
				SectionId: sectionID,
				Title:     title,
			}
			if desc != "" {
				body.Description = &desc
			}
			if publishedAt != "" {
				body.PublishedAt = &publishedAt
			}
			if submissionStart != "" {
				body.SubmissionStartAt = &submissionStart
			}
			if submissionDue != "" {
				body.SubmissionDueAt = &submissionDue
			}
			if major {
				body.IsMajor = &major
			}
			if requiresTeam {
				body.RequiresTeam = &requiresTeam
			}
			data, err := api.ParseResponseRaw(c.CreateHomework(api.Ctx(), nil, body))
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
	cmd.Flags().BoolVar(&requiresTeam, "requires-team", false, "Requires a team submission")
	return cmd
}

type myHomeworkListOpts struct {
	sectionID string
	done      bool
	pending   bool
	before    string
	after     string
	page      int
	limit     int
}

// NewCmdMyHomework returns personal homework commands (list + complete).
func NewCmdMyHomework() *cobra.Command {
	var opts myHomeworkListOpts
	cmd := &cobra.Command{
		Use:   "homework [command]",
		Short: "View and manage your homeworks",
		Long:  "List your assigned homeworks and mark them as complete.\nWhen no --section-id is given, aggregates homework from all your subscribed sections.",
		Example: `  # List all your homeworks (from subscribed sections)
  life-ustc me homework

  # Show only pending homeworks
  life-ustc me homework --pending

  # Filter to a specific section
  life-ustc me homework list --section-id <id>

  # Mark a homework as done
  life-ustc me homework complete <homework-id>`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMyHomeworkList(cmd, opts)
		},
	}
	addMyHomeworkListFlags(cmd, &opts)
	cmd.AddCommand(newCmdMyList())
	cmd.AddCommand(newCmdComplete())
	return cmd
}

func newCmdMyList() *cobra.Command {
	var opts myHomeworkListOpts
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List your homeworks",
		Example: `  life-ustc me homework list
  life-ustc me homework list --section-id <id>
  life-ustc me homework list --pending
  life-ustc me homework list --before 2025-06-01`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMyHomeworkList(cmd, opts)
		},
	}
	addMyHomeworkListFlags(cmd, &opts)
	return cmd
}

func addMyHomeworkListFlags(cmd *cobra.Command, opts *myHomeworkListOpts) {
	cmd.Flags().StringVar(&opts.sectionID, "section-id", "", "Section ID (required)")
	cmd.Flags().BoolVar(&opts.done, "done", false, "Show only completed homeworks")
	cmd.Flags().BoolVar(&opts.pending, "pending", false, "Show only pending homeworks")
	cmd.Flags().StringVar(&opts.before, "before", "", "Show homeworks due before this date (ISO 8601)")
	cmd.Flags().StringVar(&opts.after, "after", "", "Show homeworks due after this date (ISO 8601)")
	cmdutil.AddListFlags(cmd, &opts.page, &opts.limit)
}

func runMyHomeworkList(cmd *cobra.Command, opts myHomeworkListOpts) error {
	c, err := api.NewTypedClient(cmdutil.ServerFromCmd(cmd), true)
	if err != nil {
		return err
	}

	var data any
	var rows []map[string]any

	if opts.sectionID != "" {
		// Single section — use /api/homeworks with sectionId filter
		params := &openapi.ListHomeworksParams{
			SectionId: &opts.sectionID,
		}
		data, err = api.ParseResponseRaw(c.ListHomeworks(api.Ctx(), params))
		if err != nil {
			return err
		}
		_, rows, _, _ = cmdutil.ExtractList(data, "homeworks")
	} else {
		// All subscribed sections — use the combined endpoint
		data, err = api.ParseResponseRaw(c.GetSubscribedHomeworks(api.Ctx()))
		if err != nil {
			return err
		}
		_, rows, _, _ = cmdutil.ExtractList(data, "homeworks")

		// Client-side filtering for the combined endpoint
		if opts.done || opts.pending || opts.before != "" || opts.after != "" {
			var filtered []map[string]any
			for _, row := range rows {
				if opts.done {
					if v, _ := row["isCompleted"].(bool); !v {
						continue
					}
				}
				if opts.pending {
					if v, _ := row["isCompleted"].(bool); v {
						continue
					}
				}
				if opts.before != "" {
					if due, _ := row["submissionDueAt"].(string); due == "" || due > opts.before {
						continue
					}
				}
				if opts.after != "" {
					if due, _ := row["submissionDueAt"].(string); due == "" || due < opts.after {
						continue
					}
				}
				filtered = append(filtered, row)
			}
			rows = filtered
		}
	}

	// For JSON/JQ output, return the full response
	if output.IsJSON() {
		output.JSON(data)
		return nil
	}

	if len(rows) == 0 {
		output.Dim("  No homeworks found.")
		if opts.done || opts.pending || opts.before != "" || opts.after != "" {
			output.Hint("try adjusting your filters, or run without filters to see all items")
		}
		return nil
	}

	output.Dim(fmt.Sprintf("  %d homework(s)", len(rows)))
	output.Table(rows, []output.Column{
		{Header: "ID", Key: "id"},
		{Header: "Title", Key: "title"},
		{Header: "Section", Key: "section.code"},
		{Header: "Due", Key: "submissionDueAt"},
		{Header: "Major", Key: "isMajor"},
	})
	return nil
}

func newCmdList() *cobra.Command {
	var (
		sectionID      string
		includeDeleted bool
		page, limit    int
	)
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List homeworks for a section",
		RunE: func(cmd *cobra.Command, args []string) error {
			if sectionID == "" {
				return fmt.Errorf("--section-id is required")
			}
			c, err := api.NewTypedClient(cmdutil.ServerFromCmd(cmd), false)
			if err != nil {
				return err
			}
			inclDel := openapi.ListHomeworksParamsIncludeDeleted("false")
			if includeDeleted {
				inclDel = openapi.ListHomeworksParamsIncludeDeletedTrue
			}
			params := &openapi.ListHomeworksParams{
				SectionId:      &sectionID,
				IncludeDeleted: &inclDel,
			}
			data, err := api.ParseResponseRaw(c.ListHomeworks(api.Ctx(), params))
			if err != nil {
				return err
			}
			_, rows, total, pg := cmdutil.ExtractList(data, "homeworks")
			output.OutputList(data, rows, []output.Column{
				{Header: "ID", Key: "id"},
				{Header: "Title", Key: "title"},
				{Header: "Due", Key: "submissionDueAt"},
				{Header: "Major", Key: "isMajor"},
			}, total, pg)
			return nil
		},
	}
	cmd.Flags().StringVar(&sectionID, "section-id", "", "Section ID (required)")
	cmd.Flags().BoolVar(&includeDeleted, "include-deleted", false, "Include deleted")
	cmdutil.AddListFlags(cmd, &page, &limit)
	return cmd
}

func newCmdCreate() *cobra.Command {
	var (
		sectionID, title, desc, publishedAt, submissionStart, submissionDue string
		major, requiresTeam                                                 bool
	)
	cmd := &cobra.Command{
		Use:     "create",
		Aliases: []string{"new"},
		Short:   "Create a homework",
		Long:    "Create a homework. Prompts interactively when --section-id/--title are omitted.",
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
			c, err := api.NewTypedClient(cmdutil.ServerFromCmd(cmd), true)
			if err != nil {
				return err
			}
			body := openapi.CreateHomeworkJSONRequestBody{
				SectionId: sectionID,
				Title:     title,
			}
			if desc != "" {
				body.Description = &desc
			}
			if publishedAt != "" {
				body.PublishedAt = &publishedAt
			}
			if submissionStart != "" {
				body.SubmissionStartAt = &submissionStart
			}
			if submissionDue != "" {
				body.SubmissionDueAt = &submissionDue
			}
			if major {
				body.IsMajor = &major
			}
			if requiresTeam {
				body.RequiresTeam = &requiresTeam
			}
			data, err := api.ParseResponseRaw(c.CreateHomework(api.Ctx(), nil, body))
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
	cmd.Flags().BoolVar(&requiresTeam, "requires-team", false, "Requires a team submission")
	return cmd
}

func newCmdUpdate() *cobra.Command {
	var (
		title, publishedAt, submissionStart, submissionDue string
		major, notMajor, requiresTeam, noTeam             bool
	)
	cmd := &cobra.Command{
		Use:   "update <homework-id>",
		Short: "Update a homework",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := api.NewTypedClient(cmdutil.ServerFromCmd(cmd), true)
			if err != nil {
				return err
			}
			body := openapi.UpdateHomeworkJSONRequestBody{}
			hasUpdate := false
			if title != "" {
				body.Title = &title
				hasUpdate = true
			}
			if publishedAt != "" {
				body.PublishedAt = &publishedAt
				hasUpdate = true
			}
			if submissionStart != "" {
				body.SubmissionStartAt = &submissionStart
				hasUpdate = true
			}
			if submissionDue != "" {
				body.SubmissionDueAt = &submissionDue
				hasUpdate = true
			}
			if major {
				t := true
				body.IsMajor = &t
				hasUpdate = true
			}
			if notMajor {
				f := false
				body.IsMajor = &f
				hasUpdate = true
			}
			if requiresTeam {
				t := true
				body.RequiresTeam = &t
				hasUpdate = true
			}
			if noTeam {
				f := false
				body.RequiresTeam = &f
				hasUpdate = true
			}
			if !hasUpdate {
				return fmt.Errorf("nothing to update — specify at least one flag")
			}
			_, err = api.ParseResponseRaw(c.UpdateHomework(api.Ctx(), args[0], body))
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
	cmd.Flags().BoolVar(&requiresTeam, "requires-team", false, "Mark as requiring team")
	cmd.Flags().BoolVar(&noTeam, "no-team", false, "Mark as not requiring team")
	return cmd
}

func newCmdDelete() *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:     "delete <homework-id>",
		Aliases: []string{"rm"},
		Short:   "Delete a homework (soft delete)",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !yes {
				fmt.Print("Delete this homework? (y/N) ")
				s := bufio.NewScanner(os.Stdin)
				if s.Scan() && strings.ToLower(strings.TrimSpace(s.Text())) != "y" {
					output.Warning("Cancelled.")
					return nil
				}
			}
			c, err := api.NewTypedClient(cmdutil.ServerFromCmd(cmd), true)
			if err != nil {
				return err
			}
			_, err = api.ParseResponseRaw(c.DeleteHomework(api.Ctx(), args[0], openapi.DeleteHomeworkJSONRequestBody{}))
			if err != nil {
				return err
			}
			output.Success("Homework deleted.")
			return nil
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation")
	return cmd
}

func newCmdComplete() *cobra.Command {
	var undo bool
	cmd := &cobra.Command{
		Use:   "complete <homework-id>",
		Short: "Mark homework as complete",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := api.NewTypedClient(cmdutil.ServerFromCmd(cmd), true)
			if err != nil {
				return err
			}
			body := openapi.SetHomeworkCompletionJSONRequestBody{Completed: !undo}
			_, err = api.ParseResponseRaw(c.SetHomeworkCompletion(api.Ctx(), args[0], body))
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

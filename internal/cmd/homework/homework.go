package homework

import (
	"bufio"
	"errors"
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
			c, err := api.NewClient(cmdutil.ServerFromCmd(cmd), false)
			if err != nil {
				return err
			}
			params := url.Values{"sectionId": {args[0]}}
			if includeDeleted {
				params.Set("includeDeleted", "true")
			}
			cmdutil.ApplyListParams(params, page, limit)
			data, err := c.Get("/api/homeworks", params)
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
		major                                                    bool
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
	c, err := api.NewClient(cmdutil.ServerFromCmd(cmd), true)
	if err != nil {
		return err
	}

	// Determine section IDs to query
	var sectionIDs []string
	if opts.sectionID != "" {
		sectionIDs = []string{opts.sectionID}
	} else {
		// Fetch subscribed sections from calendar
		calData, calErr := c.Get("/api/calendar-subscriptions/current", nil)
		if calErr != nil {
			return fmt.Errorf("failed to fetch subscribed sections: %w\n\n  Tip: use --section-id to specify a section directly", calErr)
		}
		calMap := cmdutil.AsMap(calData)
		sub, _ := calMap["subscription"].(map[string]any)
		if sub == nil {
			return fmt.Errorf("no calendar subscription found\n\n  Tip: subscribe to sections first, or use --section-id")
		}
		sections, _ := sub["sections"].([]any)
		if len(sections) == 0 {
			output.Dim("  No subscribed sections — no homeworks to show.")
			output.Hint("subscribe to sections first, or use --section-id")
			return nil
		}
		for _, s := range sections {
			if sm, ok := s.(map[string]any); ok {
				if id, ok := sm["id"].(float64); ok {
					sectionIDs = append(sectionIDs, cmdutil.Itoa(int(id)))
				}
			}
		}
		if len(sectionIDs) == 0 {
			output.Dim("  No subscribed sections — no homeworks to show.")
			return nil
		}
		output.VerboseF("fetching homework from %d subscribed sections", len(sectionIDs))
	}

	// Build common filter params (reusable for bulk and per-section)
	filterParams := func() url.Values {
		p := url.Values{}
		if opts.done {
			p.Set("isCompleted", "true")
		}
		if opts.pending {
			p.Set("isCompleted", "false")
		}
		if opts.before != "" {
			p.Set("dueBefore", opts.before)
		}
		if opts.after != "" {
			p.Set("dueAfter", opts.after)
		}
		return p
	}

	var data any
	var rows []map[string]any
	var total int

	if len(sectionIDs) == 1 {
		// Single section — always use sectionId
		params := filterParams()
		params.Set("sectionId", sectionIDs[0])
		cmdutil.ApplyListParams(params, opts.page, opts.limit)
		data, err = c.Get("/api/homeworks", params)
		if err != nil {
			return err
		}
		_, rows, total, _ = cmdutil.ExtractList(data, "homeworks")
	} else {
		// Multiple sections — try bulk sectionIds, fall back to per-section
		params := filterParams()
		params.Set("sectionIds", strings.Join(sectionIDs, ","))
		cmdutil.ApplyListParams(params, opts.page, opts.limit)
		data, err = c.Get("/api/homeworks", params)

		var apiErr *api.APIError
		if err != nil && errors.As(err, &apiErr) && apiErr.Status == 400 {
			// Server doesn't support sectionIds yet — fall back to per-section queries
			output.VerboseF("bulk sectionIds not supported, falling back to per-section queries")
			var allRows []map[string]any
			for _, sid := range sectionIDs {
				fp := filterParams()
				fp.Set("sectionId", sid)
				sData, sErr := c.Get("/api/homeworks", fp)
				if sErr != nil {
					output.VerboseF("section %s: %s", sid, sErr)
					continue
				}
				_, sRows, _, _ := cmdutil.ExtractList(sData, "homeworks")
				allRows = append(allRows, sRows...)
			}
			rows = allRows
			total = len(allRows)
			// Build a synthetic response for JSON output
			data = map[string]any{"homeworks": allRows}
		} else if err != nil {
			return err
		} else {
			_, rows, total, _ = cmdutil.ExtractList(data, "homeworks")
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

	if total > 0 {
		output.Dim(fmt.Sprintf("  %d homework(s) across %d section(s)", total, len(sectionIDs)))
	} else {
		output.Dim(fmt.Sprintf("  %d homework(s) across %d section(s)", len(rows), len(sectionIDs)))
	}
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
			c, err := api.NewClient(cmdutil.ServerFromCmd(cmd), false)
			if err != nil {
				return err
			}
			params := url.Values{"sectionId": {sectionID}}
			if includeDeleted {
				params.Set("includeDeleted", "true")
			}
			cmdutil.ApplyListParams(params, page, limit)
			data, err := c.Get("/api/homeworks", params)
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
		major                                                               bool
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
				return fmt.Errorf("nothing to update — specify at least one flag")
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

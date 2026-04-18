package comment

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

func promptSelect(label string, choices []string) string {
	fmt.Printf("%s:\n", label)
	for i, c := range choices {
		fmt.Printf("  %d) %s\n", i+1, c)
	}
	fmt.Print("Choice: ")
	s := bufio.NewScanner(os.Stdin)
	if s.Scan() {
		text := strings.TrimSpace(s.Text())
		for i, c := range choices {
			if text == cmdutil.Itoa(i+1) || strings.EqualFold(text, c) {
				return c
			}
		}
	}
	return choices[0]
}

var targetTypes = []string{"section", "course", "teacher", "section-teacher", "homework"}

func NewCmdComment() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "comment <command>",
		Short: "Read and write comments",
	}
	cmd.AddCommand(newCmdList())
	cmd.AddCommand(newCmdView())
	cmd.AddCommand(newCmdCreate())
	cmd.AddCommand(newCmdUpdate())
	cmd.AddCommand(newCmdDelete())
	cmd.AddCommand(newCmdReact())
	return cmd
}

func newCmdList() *cobra.Command {
	var targetType, targetID, sectionID, teacherID string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List comments for a target",
		RunE: func(cmd *cobra.Command, args []string) error {
			if targetType == "" {
				return fmt.Errorf("--target-type is required")
			}
			c, err := api.NewClient(cmdutil.ServerFromCmd(cmd), false)
			if err != nil {
				return err
			}
			params := url.Values{"targetType": {targetType}}
			if targetID != "" {
				params.Set("targetId", targetID)
			}
			if sectionID != "" {
				params.Set("sectionId", sectionID)
			}
			if teacherID != "" {
				params.Set("teacherId", teacherID)
			}
			data, err := c.Get("/api/comments", params)
			if err != nil {
				return err
			}
			_, rows, total, pg := cmdutil.ExtractList(data, "comments")
			output.OutputList(data, rows, []output.Column{
				{Header: "Body", Key: "body"},
				{Header: "Visibility", Key: "visibility"},
				{Header: "Created", Key: "createdAt"},
			}, total, pg)
			return nil
		},
	}
	cmd.Flags().StringVar(&targetType, "target-type", "", "Target type (section, course, teacher, section-teacher, homework)")
	cmd.Flags().StringVar(&targetID, "target-id", "", "Target ID")
	cmd.Flags().StringVar(&sectionID, "section-id", "", "Section ID (for section-teacher)")
	cmd.Flags().StringVar(&teacherID, "teacher-id", "", "Teacher ID (for section-teacher)")
	return cmd
}

func newCmdView() *cobra.Command {
	return &cobra.Command{
		Use:   "view <comment-id>",
		Short: "View a comment thread",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := api.NewClient(cmdutil.ServerFromCmd(cmd), false)
			if err != nil {
				return err
			}
			data, err := c.Get(fmt.Sprintf("/api/comments/%s", args[0]), nil)
			if err != nil {
				return err
			}
			if output.IsJSON() {
				output.JSON(data)
				return nil
			}
			m := cmdutil.AsMap(data)
			output.KVWithTitle([]output.KVPair{
				{Key: "ID", Value: output.Resolve(m, "id")},
				{Key: "Body", Value: output.Resolve(m, "body")},
				{Key: "Visibility", Value: output.Resolve(m, "visibility")},
				{Key: "Anonymous", Value: output.Resolve(m, "isAnonymous")},
				{Key: "Created", Value: output.Resolve(m, "createdAt")},
				{Key: "Updated", Value: output.Resolve(m, "updatedAt")},
			}, "Comment")

			if replies, ok := m["replies"].([]any); ok && len(replies) > 0 {
				fmt.Println()
				output.Bold("  Replies")
				var rows []map[string]any
				for _, r := range replies {
					if row, ok := r.(map[string]any); ok {
						rows = append(rows, row)
					}
				}
				output.Table(rows, []output.Column{
					{Header: "Body", Key: "body"},
					{Header: "Created", Key: "createdAt"},
				})
			}
			return nil
		},
	}
}

func newCmdCreate() *cobra.Command {
	var (
		targetType, targetID, sectionID, teacherID string
		body, visibility, parentID                 string
		anonymous                                  bool
	)
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Post a comment",
		Long:  "Post a comment. Prompts interactively when --target-type/--body are omitted.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if targetType == "" || body == "" {
				if !isInteractive() {
					return fmt.Errorf("--target-type and --body are required in non-interactive mode")
				}
				if targetType == "" {
					targetType = promptSelect("Target type", targetTypes)
				}
				if targetType == "section-teacher" {
					if sectionID == "" {
						sectionID = promptText("Section ID")
					}
					if teacherID == "" {
						teacherID = promptText("Teacher ID")
					}
				} else if targetID == "" {
					targetID = promptText("Target ID")
				}
				if body == "" {
					body = promptText("Comment body")
				}
			}

			// Validate required IDs
			if targetType == "section-teacher" {
				if sectionID == "" || teacherID == "" {
					return fmt.Errorf("--section-id and --teacher-id are required for section-teacher target")
				}
			} else if targetID == "" {
				return fmt.Errorf("--target-id is required for this target type")
			}

			c, err := api.NewClient(cmdutil.ServerFromCmd(cmd), true)
			if err != nil {
				return err
			}
			payload := map[string]any{
				"targetType":  targetType,
				"body":        body,
				"visibility":  visibility,
				"isAnonymous": anonymous,
			}
			if targetID != "" {
				payload["targetId"] = targetID
			}
			if sectionID != "" {
				payload["sectionId"] = sectionID
			}
			if teacherID != "" {
				payload["teacherId"] = teacherID
			}
			if parentID != "" {
				payload["parentId"] = parentID
			}
			data, err := c.Post("/api/comments", payload)
			if err != nil {
				return err
			}
			m := cmdutil.AsMap(data)
			id, _ := m["id"].(string)
			output.Success(fmt.Sprintf("Comment created: %s", id))
			return nil
		},
	}
	cmd.Flags().StringVar(&targetType, "target-type", "", "Target type")
	cmd.Flags().StringVar(&targetID, "target-id", "", "Target ID")
	cmd.Flags().StringVar(&sectionID, "section-id", "", "Section ID")
	cmd.Flags().StringVar(&teacherID, "teacher-id", "", "Teacher ID")
	cmd.Flags().StringVarP(&body, "body", "b", "", "Comment body")
	cmd.Flags().StringVar(&visibility, "visibility", "public", "Visibility (public, logged_in_only, anonymous)")
	cmd.Flags().BoolVar(&anonymous, "anonymous", false, "Post anonymously")
	cmd.Flags().StringVar(&parentID, "parent-id", "", "Reply to comment ID")
	return cmd
}

func newCmdUpdate() *cobra.Command {
	var body, visibility string
	cmd := &cobra.Command{
		Use:   "update <comment-id>",
		Short: "Edit a comment",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := api.NewClient(cmdutil.ServerFromCmd(cmd), true)
			if err != nil {
				return err
			}
			payload := map[string]any{}
			if body != "" {
				payload["body"] = body
			}
			if visibility != "" {
				payload["visibility"] = visibility
			}
			if len(payload) == 0 {
				return fmt.Errorf("nothing to update")
			}
			_, err = c.Patch(fmt.Sprintf("/api/comments/%s", args[0]), payload)
			if err != nil {
				return err
			}
			output.Success("Comment updated.")
			return nil
		},
	}
	cmd.Flags().StringVarP(&body, "body", "b", "", "New body")
	cmd.Flags().StringVar(&visibility, "visibility", "", "Visibility")
	return cmd
}

func newCmdDelete() *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "delete <comment-id>",
		Short: "Delete a comment",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !yes {
				fmt.Print("Delete this comment? (y/N) ")
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
			_, err = c.Delete(fmt.Sprintf("/api/comments/%s", args[0]), nil)
			if err != nil {
				return err
			}
			output.Success("Comment deleted.")
			return nil
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip confirmation")
	return cmd
}

func newCmdReact() *cobra.Command {
	var reactionType string
	var remove bool
	cmd := &cobra.Command{
		Use:   "react <comment-id>",
		Short: "Add or remove a reaction",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := api.NewClient(cmdutil.ServerFromCmd(cmd), true)
			if err != nil {
				return err
			}
			if remove {
				_, err = c.Delete(fmt.Sprintf("/api/comments/%s/reactions", args[0]), url.Values{"type": {reactionType}})
				if err != nil {
					return err
				}
				output.Success("Reaction removed.")
			} else {
				_, err = c.Post(fmt.Sprintf("/api/comments/%s/reactions", args[0]), map[string]any{"type": reactionType})
				if err != nil {
					return err
				}
				output.Success("Reaction added.")
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&reactionType, "type", "", "Reaction type/emoji (required)")
	cmd.MarkFlagRequired("type")
	cmd.Flags().BoolVar(&remove, "remove", false, "Remove reaction")
	return cmd
}

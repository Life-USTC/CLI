package admin

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Life-USTC/CLI/internal/api"
	"github.com/Life-USTC/CLI/internal/cmd/cmdutil"
	"github.com/Life-USTC/CLI/internal/output"
)

func NewCmdAdmin() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "admin <command>",
		Short: "Admin operations (requires admin privileges)",
	}
	cmd.AddCommand(newCmdUser())
	cmd.AddCommand(newCmdSuspension())
	cmd.AddCommand(newCmdComment())
	cmd.AddCommand(newCmdDescription())
	cmd.AddCommand(newCmdHomework())
	return cmd
}

// --- User ---

func newCmdUser() *cobra.Command {
	cmd := &cobra.Command{Use: "user <command>", Short: "Manage users"}
	cmd.AddCommand(newCmdUserList())
	cmd.AddCommand(newCmdUserUpdate())
	return cmd
}

func newCmdUserList() *cobra.Command {
	var (
		search      string
		page, limit int
	)
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List users",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := api.NewClient(cmdutil.ServerFromCmd(cmd), true)
			if err != nil {
				return err
			}
			params := url.Values{}
			if search != "" {
				params.Set("search", search)
			}
			if page > 0 {
				params.Set("page", cmdutil.Itoa(page))
			}
			if limit > 0 {
				params.Set("limit", cmdutil.Itoa(limit))
			}
			data, err := c.Get("/api/admin/users", params)
			if err != nil {
				return err
			}
			_, rows, total, pg := cmdutil.ExtractList(data)
			output.OutputList(data, rows, []output.Column{
				{Header: "Name", Key: "name"},
				{Header: "Email", Key: "email"},
				{Header: "Username", Key: "username"},
				{Header: "Admin", Key: "isAdmin"},
			}, total, pg)
			return nil
		},
	}
	cmd.Flags().StringVar(&search, "search", "", "Search")
	cmd.Flags().IntVar(&page, "page", 0, "Page")
	cmd.Flags().IntVar(&limit, "limit", 0, "Limit")
	return cmd
}

func newCmdUserUpdate() *cobra.Command {
	var (
		name, username string
		admin, noAdmin bool
	)
	cmd := &cobra.Command{
		Use:   "update <user-id>",
		Short: "Update a user",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := api.NewClient(cmdutil.ServerFromCmd(cmd), true)
			if err != nil {
				return err
			}
			body := map[string]any{}
			if name != "" {
				body["name"] = name
			}
			if username != "" {
				body["username"] = username
			}
			if admin {
				body["isAdmin"] = true
			}
			if noAdmin {
				body["isAdmin"] = false
			}
			if len(body) == 0 {
				return fmt.Errorf("nothing to update")
			}
			_, err = c.Patch(fmt.Sprintf("/api/admin/users/%s", args[0]), body)
			if err != nil {
				return err
			}
			output.Success("User updated.")
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Name")
	cmd.Flags().StringVar(&username, "username", "", "Username")
	cmd.Flags().BoolVar(&admin, "admin", false, "Set admin")
	cmd.Flags().BoolVar(&noAdmin, "no-admin", false, "Remove admin")
	return cmd
}

// --- Suspension ---

func newCmdSuspension() *cobra.Command {
	cmd := &cobra.Command{Use: "suspension <command>", Short: "Manage suspensions"}
	cmd.AddCommand(newCmdSuspensionList())
	cmd.AddCommand(newCmdSuspensionCreate())
	cmd.AddCommand(newCmdSuspensionLift())
	return cmd
}

func newCmdSuspensionList() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List suspensions",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := api.NewClient(cmdutil.ServerFromCmd(cmd), true)
			if err != nil {
				return err
			}
			data, err := c.Get("/api/admin/suspensions", nil)
			if err != nil {
				return err
			}
			_, rows, total, pg := cmdutil.ExtractList(data)
			output.OutputList(data, rows, []output.Column{
				{Header: "User", Key: "user.name"},
				{Header: "Reason", Key: "reason"},
				{Header: "Expires", Key: "expiresAt"},
				{Header: "Created", Key: "createdAt"},
			}, total, pg)
			return nil
		},
	}
}

func newCmdSuspensionCreate() *cobra.Command {
	var userID, reason, note, expiresAt string
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Suspend a user",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := api.NewClient(cmdutil.ServerFromCmd(cmd), true)
			if err != nil {
				return err
			}
			body := map[string]any{"userId": userID}
			if reason != "" {
				body["reason"] = reason
			}
			if note != "" {
				body["note"] = note
			}
			if expiresAt != "" {
				body["expiresAt"] = expiresAt
			}
			_, err = c.Post("/api/admin/suspensions", body)
			if err != nil {
				return err
			}
			output.Success("User suspended.")
			return nil
		},
	}
	cmd.Flags().StringVar(&userID, "user-id", "", "User ID (required)")
	_ = cmd.MarkFlagRequired("user-id")
	cmd.Flags().StringVar(&reason, "reason", "", "Reason")
	cmd.Flags().StringVar(&note, "note", "", "Note")
	cmd.Flags().StringVar(&expiresAt, "expires-at", "", "Expiry date (ISO 8601)")
	return cmd
}

func newCmdSuspensionLift() *cobra.Command {
	return &cobra.Command{
		Use:   "lift <suspension-id>",
		Short: "Lift a suspension",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := api.NewClient(cmdutil.ServerFromCmd(cmd), true)
			if err != nil {
				return err
			}
			_, err = c.Patch(fmt.Sprintf("/api/admin/suspensions/%s", args[0]), map[string]any{"lifted": true})
			if err != nil {
				return err
			}
			output.Success("Suspension lifted.")
			return nil
		},
	}
}

// --- Admin Comment ---

func newCmdComment() *cobra.Command {
	cmd := &cobra.Command{Use: "comment <command>", Short: "Moderate comments"}
	cmd.AddCommand(newCmdCommentList())
	cmd.AddCommand(newCmdCommentModerate())
	return cmd
}

func newCmdCommentList() *cobra.Command {
	var status string
	var limit int
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List comments (admin)",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := api.NewClient(cmdutil.ServerFromCmd(cmd), true)
			if err != nil {
				return err
			}
			params := url.Values{}
			if status != "" {
				params.Set("status", status)
			}
			if limit > 0 {
				params.Set("limit", cmdutil.Itoa(limit))
			}
			data, err := c.Get("/api/admin/comments", params)
			if err != nil {
				return err
			}
			_, rows, total, pg := cmdutil.ExtractList(data)
			output.OutputList(data, rows, []output.Column{
				{Header: "Body", Key: "body"},
				{Header: "User", Key: "user.name"},
				{Header: "Status", Key: "status"},
				{Header: "Created", Key: "createdAt"},
			}, total, pg)
			return nil
		},
	}
	cmd.Flags().StringVar(&status, "status", "", "Status filter (active, softbanned, deleted, suspended)")
	cmd.Flags().IntVar(&limit, "limit", 0, "Limit")
	return cmd
}

func newCmdCommentModerate() *cobra.Command {
	var status, note string
	cmd := &cobra.Command{
		Use:   "moderate <comment-id>",
		Short: "Moderate a comment",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := api.NewClient(cmdutil.ServerFromCmd(cmd), true)
			if err != nil {
				return err
			}
			body := map[string]any{"status": status}
			if note != "" {
				body["note"] = note
			}
			_, err = c.Patch(fmt.Sprintf("/api/admin/comments/%s", args[0]), body)
			if err != nil {
				return err
			}
			output.Success("Comment moderated.")
			return nil
		},
	}
	cmd.Flags().StringVar(&status, "status", "", "Status (active, softbanned, deleted)")
	_ = cmd.MarkFlagRequired("status")
	cmd.Flags().StringVar(&note, "note", "", "Note")
	return cmd
}

// --- Admin Description ---

func newCmdDescription() *cobra.Command {
	cmd := &cobra.Command{Use: "description <command>", Short: "Moderate descriptions"}
	cmd.AddCommand(newCmdDescriptionList())
	return cmd
}

func newCmdDescriptionList() *cobra.Command {
	var targetType, hasContent, search string
	var limit int
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List descriptions (admin)",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := api.NewClient(cmdutil.ServerFromCmd(cmd), true)
			if err != nil {
				return err
			}
			params := url.Values{}
			if targetType != "" {
				params.Set("targetType", targetType)
			}
			if hasContent != "" {
				params.Set("hasContent", hasContent)
			}
			if search != "" {
				params.Set("search", search)
			}
			if limit > 0 {
				params.Set("limit", cmdutil.Itoa(limit))
			}
			data, err := c.Get("/api/admin/descriptions", params)
			if err != nil {
				return err
			}
			_, rows, total, pg := cmdutil.ExtractList(data)
			output.OutputList(data, rows, []output.Column{
				{Header: "Type", Key: "targetType"},
				{Header: "Target", Key: "targetId"},
				{Header: "Content", Key: "content"},
				{Header: "Updated", Key: "updatedAt"},
			}, total, pg)
			return nil
		},
	}
	cmd.Flags().StringVar(&targetType, "target-type", "", "Filter by type")
	cmd.Flags().StringVar(&hasContent, "has-content", "", "Filter: withContent, empty, all")
	cmd.Flags().StringVar(&search, "search", "", "Search")
	cmd.Flags().IntVar(&limit, "limit", 0, "Limit")
	return cmd
}

// --- Admin Homework ---

func newCmdHomework() *cobra.Command {
	cmd := &cobra.Command{Use: "homework <command>", Short: "Moderate homeworks"}
	cmd.AddCommand(newCmdHomeworkList())
	cmd.AddCommand(newCmdHomeworkDelete())
	return cmd
}

func newCmdHomeworkList() *cobra.Command {
	var status, search string
	var limit int
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List homeworks (admin)",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := api.NewClient(cmdutil.ServerFromCmd(cmd), true)
			if err != nil {
				return err
			}
			params := url.Values{}
			if status != "" {
				params.Set("status", status)
			}
			if search != "" {
				params.Set("search", search)
			}
			if limit > 0 {
				params.Set("limit", cmdutil.Itoa(limit))
			}
			data, err := c.Get("/api/admin/homeworks", params)
			if err != nil {
				return err
			}
			_, rows, total, pg := cmdutil.ExtractList(data)
			output.OutputList(data, rows, []output.Column{
				{Header: "Title", Key: "title"},
				{Header: "Section", Key: "section.code"},
				{Header: "Due", Key: "submissionDueAt"},
				{Header: "Status", Key: "status"},
			}, total, pg)
			return nil
		},
	}
	cmd.Flags().StringVar(&status, "status", "", "Status (all, active, deleted)")
	cmd.Flags().StringVar(&search, "search", "", "Search")
	cmd.Flags().IntVar(&limit, "limit", 0, "Limit")
	return cmd
}

func newCmdHomeworkDelete() *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "delete <homework-id>",
		Short: "Delete a homework (admin)",
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
			_, err = c.Delete(fmt.Sprintf("/api/admin/homeworks/%s", args[0]), nil)
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

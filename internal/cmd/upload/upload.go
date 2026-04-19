package upload

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Life-USTC/CLI/internal/api"
	"github.com/Life-USTC/CLI/internal/cmd/cmdutil"
	"github.com/Life-USTC/CLI/internal/output"
)

func NewCmdUpload() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "upload [command]",
		Short: "Manage file uploads",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUploadList(cmd)
		},
	}
	cmd.AddCommand(newCmdList())
	cmd.AddCommand(newCmdFile())
	cmd.AddCommand(newCmdRename())
	cmd.AddCommand(newCmdDelete())
	cmd.AddCommand(newCmdDownload())
	return cmd
}

func runUploadList(cmd *cobra.Command) error {
	c, err := api.NewClient(cmdutil.ServerFromCmd(cmd), true)
	if err != nil {
		return err
	}
	data, err := c.Get("/api/uploads", nil)
	if err != nil {
		return err
	}
	if output.IsJSON() {
		output.JSON(data)
		return nil
	}
	m := cmdutil.AsMap(data)
	_, rows, _, _ := cmdutil.ExtractList(data)

	if m != nil {
		used, _ := m["totalSize"].(float64)
		quota, _ := m["quota"].(float64)
		if quota > 0 {
			output.Dim(fmt.Sprintf("  Usage: %s / %s", humanSize(int64(used)), humanSize(int64(quota))))
		}
	}

	output.Table(rows, []output.Column{
		{Header: "Filename", Key: "filename"},
		{Header: "Type", Key: "contentType"},
		{Header: "Size", Key: "size"},
	})
	return nil
}

func newCmdList() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List your uploads",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUploadList(cmd)
		},
	}
}

func newCmdFile() *cobra.Command {
	var contentType string
	cmd := &cobra.Command{
		Use:   "file <filepath>",
		Short: "Upload a file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath := args[0]
			f, err := os.Open(filePath)
			if err != nil {
				return err
			}
			defer func() { _ = f.Close() }()

			stat, err := f.Stat()
			if err != nil {
				return err
			}

			if contentType == "" {
				contentType = guessContentType(filePath)
			}

			c, err := api.NewClient(cmdutil.ServerFromCmd(cmd), true)
			if err != nil {
				return err
			}

			// Step 1: Create upload
			createResp, err := c.Post("/api/uploads", map[string]any{
				"filename":    filepath.Base(filePath),
				"contentType": contentType,
				"size":        stat.Size(),
			})
			if err != nil {
				return err
			}
			cm := cmdutil.AsMap(createResp)
			uploadURL, _ := cm["uploadUrl"].(string)
			uploadID, _ := cm["id"].(string)

			if uploadURL == "" {
				return fmt.Errorf("server did not return an upload URL")
			}

			// Step 2: PUT to S3
			req, err := http.NewRequest("PUT", uploadURL, f)
			if err != nil {
				return err
			}
			req.Header.Set("Content-Type", contentType)
			req.ContentLength = stat.Size()

			httpClient := &http.Client{}
			resp, err := httpClient.Do(req)
			if err != nil {
				return err
			}
			_ = resp.Body.Close()
			if resp.StatusCode >= 400 {
				return fmt.Errorf("S3 upload failed with status %d", resp.StatusCode)
			}

			// Step 3: Complete
			_, err = c.Post("/api/uploads/complete", map[string]any{"id": uploadID})
			if err != nil {
				return err
			}

			output.Success(fmt.Sprintf("Uploaded %s (ID: %s)", filepath.Base(filePath), uploadID))
			return nil
		},
	}
	cmd.Flags().StringVar(&contentType, "content-type", "", "Content type")
	return cmd
}

func newCmdRename() *cobra.Command {
	var filename string
	cmd := &cobra.Command{
		Use:   "rename <upload-id>",
		Short: "Rename an upload",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := api.NewClient(cmdutil.ServerFromCmd(cmd), true)
			if err != nil {
				return err
			}
			_, err = c.Patch(fmt.Sprintf("/api/uploads/%s", args[0]), map[string]any{"filename": filename})
			if err != nil {
				return err
			}
			output.Success("Upload renamed.")
			return nil
		},
	}
	cmd.Flags().StringVar(&filename, "filename", "", "New filename (required)")
	_ = cmd.MarkFlagRequired("filename")
	return cmd
}

func newCmdDelete() *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "delete <upload-id>",
		Short: "Delete an upload",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !yes {
				fmt.Print("Delete this upload? (y/N) ")
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
			_, err = c.Delete(fmt.Sprintf("/api/uploads/%s", args[0]), nil)
			if err != nil {
				return err
			}
			output.Success("Upload deleted.")
			return nil
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip confirmation")
	return cmd
}

func newCmdDownload() *cobra.Command {
	var outFile string
	cmd := &cobra.Command{
		Use:   "download <upload-id>",
		Short: "Download a file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := api.NewClient(cmdutil.ServerFromCmd(cmd), true)
			if err != nil {
				return err
			}
			resp, err := c.GetRaw(fmt.Sprintf("/api/uploads/%s/download", args[0]), nil)
			if err != nil {
				return err
			}
			defer func() { _ = resp.Body.Close() }()
			data, err := io.ReadAll(resp.Body)
			if err != nil {
				return err
			}
			if outFile != "" {
				if err := os.WriteFile(outFile, data, 0o644); err != nil {
					return err
				}
				output.Success(fmt.Sprintf("Saved to %s", outFile))
			} else {
				_, _ = os.Stdout.Write(data)
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&outFile, "output", "o", "", "Save to file")
	return cmd
}

func humanSize(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

func guessContentType(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".pdf":
		return "application/pdf"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".txt":
		return "text/plain"
	case ".md":
		return "text/markdown"
	case ".zip":
		return "application/zip"
	default:
		return "application/octet-stream"
	}
}

// Package cmdutil provides shared helpers for all commands.
package cmdutil

import (
	"fmt"

	"github.com/Life-USTC/CLI/internal/config"
	"github.com/spf13/cobra"
)

// ServerFromCmd extracts the --server flag, falling back to the configured default.
func ServerFromCmd(cmd *cobra.Command) string {
	s, _ := cmd.Root().PersistentFlags().GetString("server")
	if s == "" {
		return config.GetDefaultServer()
	}
	return s
}

func Itoa(i int) string { return fmt.Sprintf("%d", i) }

// AddListFlags registers the standard --limit/-L and --page/-p flags on a command.
func AddListFlags(cmd *cobra.Command, page, limit *int) {
	cmd.Flags().IntVarP(limit, "limit", "L", 0, "Maximum number of results to fetch")
	cmd.Flags().IntVarP(page, "page", "p", 0, "Page number for paginated results")
}

// ApplyListParams sets page and limit on a url.Values-like map.
func ApplyListParams(params interface{ Set(string, string) }, page, limit int) {
	if page > 0 {
		params.Set("page", Itoa(page))
	}
	if limit > 0 {
		params.Set("limit", Itoa(limit))
	}
}

// ExtractList pulls rows and pagination info from a standard API list response.
func ExtractList(data any, listKeys ...string) (raw any, rows []map[string]any, total int, page int) {
	raw = data
	m, ok := data.(map[string]any)
	if !ok {
		if arr, ok := data.([]any); ok {
			rows = toRows(arr)
		}
		return
	}

	keys := listKeys
	if len(keys) == 0 {
		keys = []string{"items", "data", "results"}
	}

	for _, key := range keys {
		if list, ok := m[key]; ok {
			if arr, ok := list.([]any); ok {
				rows = toRows(arr)
				break
			}
		}
	}

	if len(rows) == 0 {
		if arr, ok := data.([]any); ok {
			rows = toRows(arr)
		}
	}

	if t, ok := m["total"].(float64); ok {
		total = int(t)
	}
	if p, ok := m["page"].(float64); ok {
		page = int(p)
	}
	return
}

func toRows(arr []any) []map[string]any {
	rows := make([]map[string]any, 0, len(arr))
	for _, item := range arr {
		if row, ok := item.(map[string]any); ok {
			rows = append(rows, row)
		}
	}
	return rows
}

// AsMap safely casts any to map[string]any.
func AsMap(v any) map[string]any {
	if m, ok := v.(map[string]any); ok {
		return m
	}
	return nil
}

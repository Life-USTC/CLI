// Package output provides gh-style pretty printing: tables, key-value,
// JSON, status messages, and script-friendly output (--jq).
package output

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/itchyny/gojq"
	"github.com/mattn/go-runewidth"
	"golang.org/x/term"
)

// Opts holds output formatting preferences set from root flags.
type Opts struct {
	Format  string // "table" or "json"
	NoColor bool
	JQ      string // jq filter expression (implies JSON)
	Verbose bool
}

var Current = &Opts{Format: "table"}

func IsJSON() bool      { return Current.Format == "json" || Current.JQ != "" }
func IsTTY() bool       { return term.IsTerminal(int(os.Stdout.Fd())) }
func Writer() io.Writer { return os.Stdout }

// --- Logging helpers ---

// Errorf prints a red ✗ error to stderr.
func Errorf(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "%s %s\n", color.RedString("✗"), fmt.Sprintf(format, a...))
}

// Hint prints a dim hint to stderr.
func Hint(msg string) {
	fmt.Fprintf(os.Stderr, "%s\n", color.New(color.Faint).Sprintf("hint: %s", msg))
}

// VerboseF prints debug info to stderr when --verbose is set.
func VerboseF(format string, a ...any) {
	if !Current.Verbose {
		return
	}
	fmt.Fprintf(os.Stderr, "%s %s\n", color.New(color.Faint).Sprint("[verbose]"), fmt.Sprintf(format, a...))
}

// --- JQ filter ---

// ApplyJQ applies a jq expression to data and prints results to stdout.
// Returns an error if the expression is invalid.
func ApplyJQ(data any, expr string) error {
	query, err := gojq.Parse(expr)
	if err != nil {
		return fmt.Errorf("invalid --jq expression: %w", err)
	}
	iter := query.Run(data)
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if err, isErr := v.(error); isErr {
			return fmt.Errorf("jq error: %w", err)
		}
		switch val := v.(type) {
		case string:
			fmt.Println(val)
		case nil:
			fmt.Println("null")
		default:
			b, _ := json.Marshal(val)
			fmt.Println(string(b))
		}
	}
	return nil
}

// --- JSON output ---

func JSON(data any) {
	if Current.JQ != "" {
		if err := ApplyJQ(data, Current.JQ); err != nil {
			Errorf("%s", err)
		}
		return
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	_ = enc.Encode(data)
}

// --- Table output ---

// Column describes a table column with a display header and a key path
// (dot-separated for nested access, e.g. "course.namePrimary").
type Column struct {
	Header string
	Key    string
}

// ansiRe strips ANSI escape sequences for width measurement.
var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// displayWidth returns the visual column width of s, ignoring ANSI codes.
func displayWidth(s string) int {
	return runewidth.StringWidth(ansiRe.ReplaceAllString(s, ""))
}

// padRight pads s to width w using display-aware padding.
func padRight(s string, w int) string {
	dw := displayWidth(s)
	if dw >= w {
		return s
	}
	return s + strings.Repeat(" ", w-dw)
}

// termWidth returns the current terminal width, or 120 as default.
func termWidth() int {
	if w, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil && w > 0 {
		return w
	}
	return 120
}

func Table(rows []map[string]any, cols []Column) {
	TableTo(os.Stdout, rows, cols, "No results.")
}

func TableTo(w io.Writer, rows []map[string]any, cols []Column, emptyMsg string) {
	if len(rows) == 0 {
		Dim(emptyMsg)
		return
	}

	gap := 2 // minimum gap between columns

	// Format all cells and headers
	headers := make([]string, len(cols))
	for i, c := range cols {
		headers[i] = color.New(color.Bold, color.Faint).Sprint(strings.ToUpper(c.Header))
	}

	formatted := make([][]string, len(rows))
	for r, row := range rows {
		formatted[r] = make([]string, len(cols))
		for i, c := range cols {
			formatted[r][i] = FormatCell(Resolve(row, c.Key))
		}
	}

	// Compute max display width per column (header + all data rows)
	colWidths := make([]int, len(cols))
	for i, c := range cols {
		colWidths[i] = runewidth.StringWidth(strings.ToUpper(c.Header))
	}
	for _, cells := range formatted {
		for i, cell := range cells {
			if cw := displayWidth(cell); cw > colWidths[i] {
				colWidths[i] = cw
			}
		}
	}

	// Truncate last column to fit terminal if needed
	tw := termWidth()
	totalWidth := 0
	for i, cw := range colWidths {
		totalWidth += cw
		if i < len(colWidths)-1 {
			totalWidth += gap
		}
	}
	if totalWidth > tw && len(colWidths) > 1 {
		available := tw
		for i := 0; i < len(colWidths)-1; i++ {
			available -= colWidths[i] + gap
		}
		if available < 4 {
			available = 4
		}
		colWidths[len(colWidths)-1] = available
	}

	// Print header
	hdrParts := make([]string, len(cols))
	for i := range cols {
		hdrParts[i] = padRight(headers[i], colWidths[i])
	}
	_, _ = fmt.Fprintln(w, strings.Join(hdrParts, strings.Repeat(" ", gap)))

	// Print rows
	for _, cells := range formatted {
		parts := make([]string, len(cols))
		for i, cell := range cells {
			if i == len(cols)-1 {
				// Last column: truncate if wider than allowed
				if displayWidth(cell) > colWidths[i] {
					plain := ansiRe.ReplaceAllString(cell, "")
					cell = runewidth.Truncate(plain, colWidths[i]-1, "…")
				}
				parts[i] = cell
			} else {
				parts[i] = padRight(cell, colWidths[i])
			}
		}
		_, _ = fmt.Fprintln(w, strings.Join(parts, strings.Repeat(" ", gap)))
	}
}

// --- Key-value output ---

func KV(pairs []KVPair) {
	KVWithTitle(pairs, "")
}

func KVWithTitle(pairs []KVPair, title string) {
	if title != "" {
		fmt.Println()
		Bold("  " + title)
	}

	maxKey := 0
	for _, p := range pairs {
		if len(p.Key) > maxKey {
			maxKey = len(p.Key)
		}
	}

	for _, p := range pairs {
		if p.SkipEmpty && (p.Value == nil || fmt.Sprint(p.Value) == "") {
			continue
		}
		label := color.New(color.Bold).Sprintf("  %-*s  ", maxKey+1, p.Key+":")
		fmt.Printf("%s%s\n", label, FormatCell(p.Value))
	}
}

type KVPair struct {
	Key       string
	Value     any
	SkipEmpty bool
}

// --- High-level helpers ---

func OutputList(raw any, rows []map[string]any, cols []Column, total, page int) {
	// --jq: pipe raw data through jq filter
	if Current.JQ != "" {
		if err := ApplyJQ(raw, Current.JQ); err != nil {
			Errorf("%s", err)
		}
		return
	}
	if Current.Format == "json" {
		JSON(raw)
		return
	}

	// Pagination header
	if total > 0 && len(rows) > 0 {
		limit := len(rows)
		if total > limit {
			pages := int(math.Ceil(float64(total) / float64(limit)))
			if page > 0 {
				Dim(fmt.Sprintf("  Showing %d of %d · page %d of %d", len(rows), total, page, pages))
			} else {
				Dim(fmt.Sprintf("  Showing %d of %d · use --page/-p to paginate", len(rows), total))
			}
		} else {
			Dim(fmt.Sprintf("  %d result(s)", total))
		}
	}

	// Empty state
	if len(rows) == 0 {
		if total > 0 && page > 0 {
			// We have results but this page is empty — out of bounds
			Warning(fmt.Sprintf("Page %d is out of range (total: %d results)", page, total))
			Hint("try a lower --page value")
		} else {
			Dim("  No results found.")
			Hint("try adjusting your filters, or run without filters to see all items")
		}
		return
	}

	Table(rows, cols)
}

func OutputDetail(raw any, fields []FieldDef, title string) {
	if Current.JQ != "" {
		if err := ApplyJQ(raw, Current.JQ); err != nil {
			Errorf("%s", err)
		}
		return
	}
	if Current.Format == "json" {
		JSON(raw)
		return
	}
	data, _ := raw.(map[string]any)
	pairs := make([]KVPair, 0, len(fields))
	for _, f := range fields {
		pairs = append(pairs, KVPair{
			Key:       f.Label,
			Value:     Resolve(data, f.Key),
			SkipEmpty: f.SkipEmpty,
		})
	}
	KVWithTitle(pairs, title)
}

type FieldDef struct {
	Key       string
	Label     string
	SkipEmpty bool
}

// --- Status messages ---

func Success(msg string) { fmt.Printf("%s %s\n", color.GreenString("✓"), msg) }
func Warning(msg string) { fmt.Printf("%s %s\n", color.YellowString("!"), msg) }
func Error(msg string)   { fmt.Fprintf(os.Stderr, "%s %s\n", color.RedString("✗"), msg) }
func Info(msg string)    { Dim("  " + msg) }
func Bold(msg string)    { fmt.Println(color.New(color.Bold).Sprint(msg)) }
func Dim(msg string)     { fmt.Println(color.New(color.Faint).Sprint(msg)) }

// --- Formatting helpers ---

func Resolve(m map[string]any, key string) any {
	parts := strings.Split(key, ".")
	var cur any = m
	for _, p := range parts {
		if mm, ok := cur.(map[string]any); ok {
			cur = mm[p]
			// Fallback: if "name" resolved to nil, try "nameCn" (API convention)
			if cur == nil && p == "name" {
				cur = mm["nameCn"]
			}
		} else {
			return nil
		}
	}
	return cur
}

func FormatCell(v any) string {
	if v == nil {
		return color.New(color.Faint).Sprint("-")
	}
	switch val := v.(type) {
	case bool:
		if val {
			return color.GreenString("✓")
		}
		return color.New(color.Faint).Sprint("✗")
	case float64:
		if val == float64(int64(val)) {
			return fmt.Sprintf("%d", int64(val))
		}
		return fmt.Sprintf("%g", val)
	case string:
		if t, ok := parseISO(val); ok {
			return t.Format("2006-01-02 15:04")
		}
		return val
	default:
		return fmt.Sprint(v)
	}
}

func parseISO(s string) (time.Time, bool) {
	if len(s) < 19 || s[4] != '-' || s[7] != '-' || s[10] != 'T' || s[13] != ':' {
		return time.Time{}, false
	}
	for _, layout := range []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05",
	} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

func FormatRelativeTime(s string) string {
	t, ok := parseISO(s)
	if !ok {
		return s
	}
	d := time.Since(t)
	switch {
	case d < 0:
		return formatFuture(-d)
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	case d < 30*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	default:
		return t.Format("2006-01-02")
	}
}

func formatFuture(d time.Duration) string {
	switch {
	case d < time.Minute:
		return "in <1m"
	case d < time.Hour:
		return fmt.Sprintf("in %dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("in %dh", int(d.Hours()))
	default:
		return fmt.Sprintf("in %dd", int(d.Hours()/24))
	}
}

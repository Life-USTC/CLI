package calendar

import (
	"strings"
	"testing"
)

func TestCalendarSubscriptionDetailsUseOnlyScopeGatedURL(t *testing.T) {
	const calendarURL = "https://life.example/api/calendar-feeds/private-token.ics"
	details := calendarSubscriptionDetails(map[string]any{
		"calendarUrl": calendarURL,
		"userId":      "must-not-be-used-to-build-a-url",
	})
	if got := details[0].Value; !strings.Contains(got.(string), calendarURL) {
		t.Fatalf("URL detail = %#v, want returned calendarUrl", got)
	}
}

func TestCalendarSubscriptionDetailsExplainMissingScopeGatedURL(t *testing.T) {
	for _, calendarURL := range []any{nil, ""} {
		details := calendarSubscriptionDetails(map[string]any{
			"calendarUrl": calendarURL,
			"userId":      "must-not-be-used-to-build-a-url",
		})
		got, _ := details[0].Value.(string)
		if !strings.Contains(got, "Unavailable") || !strings.Contains(got, "workspace.calendar-feed:read") {
			t.Fatalf("URL detail = %q, want clear missing-scope message", got)
		}
		if strings.Contains(got, "must-not-be-used") {
			t.Fatalf("URL detail synthesized a fallback feed: %q", got)
		}
	}
}

func TestSetRequiresSemesterID(t *testing.T) {
	for _, args := range [][]string{
		{"999999"},
		{"999999", "--semester-id", " "},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			t.Setenv("LIFE_USTC_CONFIG_DIR", t.TempDir())
			t.Setenv("LIFE_USTC_SERVER", "http://127.0.0.1:1")

			cmd := newCmdSet()
			cmd.SetArgs(args)

			err := cmd.Execute()
			if err == nil {
				t.Fatal("expected calendar set without --semester-id to fail")
			}
			if !strings.Contains(err.Error(), "--semester-id is required") {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

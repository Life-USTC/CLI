package api

import (
	"net/url"
	"strings"
	"testing"
)

func TestRedactedRequestURLHidesCalendarFeedCredential(t *testing.T) {
	requestURL, err := url.Parse("https://life.example/api/calendar-feeds/private-token.ics?token=query-secret&download=1")
	if err != nil {
		t.Fatal(err)
	}
	got := redactedRequestURL(requestURL)
	if strings.Contains(got, "private-token") || strings.Contains(got, "query-secret") {
		t.Fatalf("redacted URL leaked feed credential: %s", got)
	}
	if got != "https://life.example/api/calendar-feeds/%5Bredacted%5D.ics" {
		t.Fatalf("redacted URL = %q", got)
	}
}

func TestRedactedRequestURLLeavesOrdinaryPathsIntact(t *testing.T) {
	requestURL, err := url.Parse("https://life.example/api/workspace/subscriptions/current")
	if err != nil {
		t.Fatal(err)
	}
	if got := redactedRequestURL(requestURL); got != requestURL.String() {
		t.Fatalf("redacted URL = %q, want %q", got, requestURL)
	}
}

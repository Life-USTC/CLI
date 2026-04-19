// Package api provides the HTTP client for the Life@USTC API.
//
// typed.go bridges the generated OpenAPI client with the existing
// auth/token-refresh logic by injecting an authTransport into the
// standard http.Client used by the generated code.
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Life-USTC/CLI/internal/auth"
	"github.com/Life-USTC/CLI/internal/config"
	openapi "github.com/Life-USTC/CLI/internal/openapi"
	"github.com/Life-USTC/CLI/internal/output"
)

// authTransport implements http.RoundTripper.
// It injects the Bearer token and retries once on 401 after a token refresh.
type authTransport struct {
	server string
	cred   *config.Credential
	base   http.RoundTripper
}

func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.ensureToken()
	if t.cred != nil {
		req.Header.Set("Authorization", "Bearer "+t.cred.AccessToken)
	}

	output.VerboseF("→ %s %s", req.Method, req.URL)
	start := time.Now()

	resp, err := t.base.RoundTrip(req)
	if err != nil {
		output.VerboseF("← error: %s (%dms)", err, time.Since(start).Milliseconds())
		return nil, err
	}

	output.VerboseF("← %d %s (%dms)", resp.StatusCode, http.StatusText(resp.StatusCode), time.Since(start).Milliseconds())

	if resp.StatusCode == 401 && t.cred != nil {
		_ = resp.Body.Close()
		newCred, refreshErr := auth.RefreshToken(t.server, t.cred)
		if refreshErr == nil && newCred != nil {
			t.cred = newCred
			_ = config.SaveCredentials(t.server, newCred)

			// Clone the request with the new token
			req2 := req.Clone(req.Context())
			req2.Header.Set("Authorization", "Bearer "+t.cred.AccessToken)
			resp, err = t.base.RoundTrip(req2)
			if err != nil {
				return nil, err
			}
		}
		if resp.StatusCode == 401 {
			_ = resp.Body.Close()
			return nil, fmt.Errorf("session expired. Please run `life-ustc auth login` again")
		}
	}

	return resp, nil
}

func (t *authTransport) ensureToken() {
	if t.cred == nil || !config.IsTokenExpired(t.cred) {
		return
	}
	newCred, err := auth.RefreshToken(t.server, t.cred)
	if err != nil || newCred == nil {
		return
	}
	t.cred = newCred
	_ = config.SaveCredentials(t.server, newCred)
}

// TypedClient wraps the generated OpenAPI client with auth support.
type TypedClient struct {
	*openapi.Client
	server string
}

// NewTypedClient creates a TypedClient with auth, using the generated OpenAPI client.
func NewTypedClient(server string, requireAuth bool) (*TypedClient, error) {
	server = strings.TrimRight(server, "/")
	cred, err := config.LoadCredentials(server)
	if err != nil {
		return nil, err
	}
	if requireAuth && cred == nil {
		return nil, fmt.Errorf("not logged in. Run `life-ustc auth login` first")
	}

	transport := &authTransport{
		server: server,
		cred:   cred,
		base:   http.DefaultTransport,
	}

	httpClient := &http.Client{
		Timeout:   30 * time.Second,
		Transport: transport,
	}

	client, err := openapi.NewClient(server, openapi.WithHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}

	return &TypedClient{Client: client, server: server}, nil
}

// ParseResponse reads a response body and decodes it into the target type.
// On non-2xx, it extracts an error message.
func ParseResponse[T any](resp *http.Response, err error) (*T, error) {
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, readErr
	}

	if resp.StatusCode >= 400 {
		msg := extractErrorMessage(body, resp.StatusCode, resp.Request.Method, resp.Request.URL.Path)
		return nil, fmt.Errorf("%s", msg)
	}

	var result T
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return &result, nil
}

// ParseResponseRaw reads a response and returns it as map[string]any for
// backward compatibility with the output module.
func ParseResponseRaw(resp *http.Response, err error) (any, error) {
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, readErr
	}

	if resp.StatusCode >= 400 {
		msg := extractErrorMessage(body, resp.StatusCode, resp.Request.Method, resp.Request.URL.Path)
		return nil, fmt.Errorf("%s", msg)
	}

	ct := resp.Header.Get("Content-Type")
	if strings.HasPrefix(ct, "application/json") {
		var result any
		if err := json.Unmarshal(body, &result); err != nil {
			return nil, err
		}
		return result, nil
	}
	return string(body), nil
}

func extractErrorMessage(body []byte, status int, method, path string) string {
	var parsed map[string]any
	msg := ""
	if json.Unmarshal(body, &parsed) == nil {
		if m, ok := parsed["message"].(string); ok {
			msg = m
		} else if e, ok := parsed["error"].(string); ok {
			msg = e
		}
	}
	if msg == "" && len(body) > 0 {
		msg = string(body)
		if len(msg) > 200 {
			msg = msg[:200]
		}
	}
	if msg != "" {
		return fmt.Sprintf("%s %s → %d: %s", method, path, status, msg)
	}
	return fmt.Sprintf("%s %s → %d", method, path, status)
}

// MarshalToMap converts any struct to map[string]any via JSON round-trip.
// Useful for passing typed responses to the output module.
func MarshalToMap(v any) map[string]any {
	b, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return nil
	}
	return m
}

// MarshalToMaps converts a slice of structs to []map[string]any.
func MarshalToMaps[T any](items []T) []map[string]any {
	result := make([]map[string]any, 0, len(items))
	for _, item := range items {
		if m := MarshalToMap(item); m != nil {
			result = append(result, m)
		}
	}
	return result
}

// Ctx returns a background context (convenience for CLI commands).
func Ctx() context.Context {
	return context.Background()
}

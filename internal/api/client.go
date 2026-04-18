// Package api provides the HTTP client for the Life@USTC API.
package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Life-USTC/CLI/internal/auth"
	"github.com/Life-USTC/CLI/internal/config"
)

// Client wraps net/http with automatic auth header injection and token refresh.
type Client struct {
	Server     string
	Cred       *config.Credential
	HTTPClient *http.Client
}

// NewClient creates a client, optionally requiring auth.
func NewClient(server string, requireAuth bool) (*Client, error) {
	server = strings.TrimRight(server, "/")
	cred, err := config.LoadCredentials(server)
	if err != nil {
		return nil, err
	}
	if requireAuth && cred == nil {
		return nil, fmt.Errorf("not logged in. Run `life-ustc auth login` first")
	}

	c := &Client{
		Server: server,
		Cred:   cred,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
	c.ensureToken()
	return c, nil
}

func (c *Client) ensureToken() {
	if c.Cred == nil || !config.IsTokenExpired(c.Cred) {
		return
	}
	newCred, err := auth.RefreshToken(c.Server, c.Cred)
	if err != nil || newCred == nil {
		return
	}
	c.Cred = newCred
	_ = config.SaveCredentials(c.Server, newCred)
}

func (c *Client) headers() http.Header {
	h := make(http.Header)
	if c.Cred != nil {
		h.Set("Authorization", "Bearer "+c.Cred.AccessToken)
	}
	return h
}

// APIError represents a non-2xx response.
type APIError struct {
	Status  int
	Method  string
	Path    string
	Message string
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("%s %s → %d: %s", e.Method, e.Path, e.Status, e.Message)
	}
	return fmt.Sprintf("%s %s → %d", e.Method, e.Path, e.Status)
}

func (c *Client) do(method, path string, params url.Values, body io.Reader, contentType string) (*http.Response, error) {
	u := c.Server + path
	if len(params) > 0 {
		u += "?" + params.Encode()
	}

	req, err := http.NewRequest(method, u, body)
	if err != nil {
		return nil, err
	}
	for k, v := range c.headers() {
		req.Header[k] = v
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}

	// Retry once on 401
	if resp.StatusCode == 401 && c.Cred != nil {
		resp.Body.Close()
		newCred, err := auth.RefreshToken(c.Server, c.Cred)
		if err == nil && newCred != nil {
			c.Cred = newCred
			_ = config.SaveCredentials(c.Server, newCred)

			// Re-read body if possible
			if body != nil {
				if seeker, ok := body.(io.ReadSeeker); ok {
					seeker.Seek(0, io.SeekStart)
				}
			}

			req2, _ := http.NewRequest(method, u, body)
			for k, v := range c.headers() {
				req2.Header[k] = v
			}
			if contentType != "" {
				req2.Header.Set("Content-Type", contentType)
			}
			resp, err = c.HTTPClient.Do(req2)
			if err != nil {
				return nil, err
			}
		}
		if resp.StatusCode == 401 {
			resp.Body.Close()
			return nil, fmt.Errorf("session expired. Please run `life-ustc auth login` again")
		}
	}

	return resp, nil
}

func (c *Client) request(method, path string, params url.Values, jsonBody any) (any, error) {
	var body io.Reader
	ct := ""
	if jsonBody != nil {
		b, err := json.Marshal(jsonBody)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(b)
		ct = "application/json"
	}

	resp, err := c.do(method, path, params, body, ct)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		msg := ""
		var parsed map[string]any
		if json.Unmarshal(respBody, &parsed) == nil {
			if m, ok := parsed["message"].(string); ok {
				msg = m
			} else if e, ok := parsed["error"].(string); ok {
				msg = e
			}
		}
		if msg == "" && len(respBody) > 0 {
			msg = string(respBody)
			if len(msg) > 200 {
				msg = msg[:200]
			}
		}
		return nil, &APIError{Status: resp.StatusCode, Method: method, Path: path, Message: msg}
	}

	ct = resp.Header.Get("Content-Type")
	if strings.HasPrefix(ct, "application/json") {
		var result any
		if err := json.Unmarshal(respBody, &result); err != nil {
			return nil, err
		}
		return result, nil
	}
	return string(respBody), nil
}

// Get performs a GET request with optional query params.
func (c *Client) Get(path string, params url.Values) (any, error) {
	return c.request("GET", path, params, nil)
}

// Post performs a POST request with a JSON body.
func (c *Client) Post(path string, body any) (any, error) {
	return c.request("POST", path, nil, body)
}

// Patch performs a PATCH request with a JSON body.
func (c *Client) Patch(path string, body any) (any, error) {
	return c.request("PATCH", path, nil, body)
}

// Put performs a PUT request with a JSON body.
func (c *Client) Put(path string, body any) (any, error) {
	return c.request("PUT", path, nil, body)
}

// Delete performs a DELETE request with optional query params.
func (c *Client) Delete(path string, params url.Values) (any, error) {
	return c.request("DELETE", path, params, nil)
}

// GetRaw performs a GET and returns the raw http.Response.
func (c *Client) GetRaw(path string, params url.Values) (*http.Response, error) {
	return c.do("GET", path, params, nil, "")
}

// PostForm posts form-encoded data.
func (c *Client) PostForm(endpoint string, data url.Values) (map[string]any, error) {
	body := strings.NewReader(data.Encode())
	resp, err := c.do("POST", endpoint, nil, body, "application/x-www-form-urlencoded")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("POST %s → %d: %s", endpoint, resp.StatusCode, string(respBody))
	}

	var result map[string]any
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, err
	}
	return result, nil
}

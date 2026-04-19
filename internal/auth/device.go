package auth

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Life-USTC/CLI/internal/config"
)

// deviceAuthResponse matches the RFC 8628 device authorization response.
type deviceAuthResponse struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete"`
	ExpiresIn               int    `json:"expires_in"`
	Interval                int    `json:"interval"`
}

// registerDeviceClient registers a public OAuth client for the device code grant.
func registerDeviceClient(endpoint string) (map[string]any, error) {
	body := map[string]any{
		"client_name":                "life-ustc-cli",
		"redirect_uris":             []string{"http://localhost/callback"},
		"token_endpoint_auth_method": "none",
		"grant_types":               []string{"urn:ietf:params:oauth:grant-type:device_code", "refresh_token"},
		"response_types":            []string{"code"},
		"scope":                     "openid profile email offline_access",
	}
	data, _ := json.Marshal(body)
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Post(endpoint, "application/json", strings.NewReader(string(data)))
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("client registration failed (%d): %s", resp.StatusCode, string(b))
	}
	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode client registration response: %w", err)
	}
	return result, nil
}

// LoginDeviceCode runs the RFC 8628 Device Authorization Grant flow.
// It prints a user code for the user to enter in a browser, then polls
// the token endpoint until approved, denied, or expired.
func LoginDeviceCode(server string) (*config.Credential, error) {
	server = strings.TrimRight(server, "/")
	fmt.Printf("Logging in to %s via device code ...\n", server)

	meta, err := discoverOAuthMetadata(server)
	if err != nil {
		return nil, err
	}

	deviceEndpoint, _ := meta["device_authorization_endpoint"].(string)
	tokenEndpoint, _ := meta["token_endpoint"].(string)
	regEndpoint, _ := meta["registration_endpoint"].(string)

	if deviceEndpoint == "" {
		return nil, fmt.Errorf("server does not support device authorization (no device_authorization_endpoint in metadata)")
	}
	if regEndpoint == "" {
		return nil, fmt.Errorf("server does not advertise a registration_endpoint")
	}

	// Register client
	clientInfo, err := registerDeviceClient(regEndpoint)
	if err != nil {
		return nil, err
	}
	clientID, _ := clientInfo["client_id"].(string)

	// Request device code
	httpClient := &http.Client{Timeout: 15 * time.Second}
	deviceResp, err := httpClient.PostForm(deviceEndpoint, url.Values{
		"client_id": {clientID},
		"scope":     {"openid profile email offline_access"},
	})
	if err != nil {
		return nil, fmt.Errorf("device authorization request failed: %w", err)
	}
	defer func() { _ = deviceResp.Body.Close() }()

	if deviceResp.StatusCode != 200 {
		b, _ := io.ReadAll(deviceResp.Body)
		return nil, fmt.Errorf("device authorization failed (%d): %s", deviceResp.StatusCode, string(b))
	}

	var devAuth deviceAuthResponse
	if err := json.NewDecoder(deviceResp.Body).Decode(&devAuth); err != nil {
		return nil, fmt.Errorf("failed to decode device authorization response: %w", err)
	}

	// Display instructions to user
	fmt.Println()
	fmt.Println("To sign in, visit:")
	if devAuth.VerificationURIComplete != "" {
		fmt.Printf("  %s\n", devAuth.VerificationURIComplete)
	} else {
		fmt.Printf("  %s\n", devAuth.VerificationURI)
	}
	fmt.Println()
	fmt.Printf("And enter the code: %s\n\n", devAuth.UserCode)

	// Try to open browser
	if devAuth.VerificationURIComplete != "" {
		_ = openBrowser(devAuth.VerificationURIComplete)
	} else {
		_ = openBrowser(devAuth.VerificationURI)
	}

	fmt.Println("Waiting for authorization...")

	// Poll token endpoint
	interval := time.Duration(devAuth.Interval) * time.Second
	if interval < 5*time.Second {
		interval = 5 * time.Second
	}
	deadline := time.Now().Add(time.Duration(devAuth.ExpiresIn) * time.Second)

	for time.Now().Before(deadline) {
		time.Sleep(interval)

		tokenData := url.Values{
			"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
			"client_id":   {clientID},
			"device_code": {devAuth.DeviceCode},
		}

		tokenResp, err := httpClient.PostForm(tokenEndpoint, tokenData)
		if err != nil {
			return nil, fmt.Errorf("token poll failed: %w", err)
		}

		tokenBody, _ := io.ReadAll(tokenResp.Body)
		_ = tokenResp.Body.Close()

		if tokenResp.StatusCode == 200 {
			var tokens map[string]any
			if err := json.Unmarshal(tokenBody, &tokens); err != nil {
				return nil, fmt.Errorf("failed to decode token response: %w", err)
			}

			now := float64(time.Now().Unix())
			expiresIn := 3600.0
			if ei, ok := tokens["expires_in"].(float64); ok {
				expiresIn = ei
			}

			return &config.Credential{
				ClientID:     clientID,
				AccessToken:  tokens["access_token"].(string),
				RefreshToken: strDefault(tokens["refresh_token"]),
				TokenType:    strDefault(tokens["token_type"]),
				ExpiresAt:    now + expiresIn,
				Scope:        strDefault(tokens["scope"]),
				Resource:     server,
			}, nil
		}

		// Parse error response
		var errResp struct {
			Error string `json:"error"`
		}
		_ = json.Unmarshal(tokenBody, &errResp)

		switch errResp.Error {
		case "authorization_pending":
			// keep polling
			continue
		case "slow_down":
			interval += 5 * time.Second
			continue
		case "expired_token":
			return nil, fmt.Errorf("device code expired — please try again")
		case "access_denied":
			return nil, fmt.Errorf("authorization denied by user")
		default:
			return nil, fmt.Errorf("token request failed: %s (HTTP %d)", string(tokenBody), tokenResp.StatusCode)
		}
	}

	return nil, fmt.Errorf("device code expired — please try again")
}

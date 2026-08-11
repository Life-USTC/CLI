package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/Life-USTC/CLI/internal/config"
	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
)

func TestRegisterPublicClientUsesNativeApplicationType(t *testing.T) {
	requests := make(chan map[string]any, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		requests <- body
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"client_id":"client-1"}`))
	}))
	t.Cleanup(server.Close)

	_, err := registerPublicClient(
		server.URL,
		[]string{"openid", "profile", "workspace.todo:read"},
		[]string{"http://127.0.0.1:46289/callback"},
		[]string{"authorization_code", "refresh_token"},
		[]string{"code"},
	)
	if err != nil {
		t.Fatal(err)
	}
	body := <-requests
	if body["application_type"] != "native" {
		t.Fatalf("application_type = %#v, want native", body["application_type"])
	}
	if body["scope"] != "openid profile workspace.todo:read" {
		t.Fatalf("scope = %#v", body["scope"])
	}
	redirectURIs, ok := body["redirect_uris"].([]any)
	if !ok || len(redirectURIs) != 1 || redirectURIs[0] != "http://127.0.0.1:46289/callback" {
		t.Fatalf("redirect_uris = %#v", body["redirect_uris"])
	}
}

func TestRegisterPublicClientOmitsUnusedDeviceRedirectMetadata(t *testing.T) {
	requests := make(chan map[string]any, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		requests <- body
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"client_id":"device-client"}`))
	}))
	t.Cleanup(server.Close)

	_, err := registerPublicClient(
		server.URL,
		[]string{"openid", "profile", "workspace.todo:read"},
		nil,
		[]string{"urn:ietf:params:oauth:grant-type:device_code", "refresh_token"},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	body := <-requests
	if body["application_type"] != "native" {
		t.Fatalf("application_type = %#v, want native", body["application_type"])
	}
	if _, ok := body["redirect_uris"]; ok {
		t.Fatalf("redirect_uris should be omitted, got %#v", body["redirect_uris"])
	}
	if _, ok := body["response_types"]; ok {
		t.Fatalf("response_types should be omitted, got %#v", body["response_types"])
	}
}

func TestOAuthScopesFromMetadata(t *testing.T) {
	scopes, err := oauthScopesFromMetadata(map[string]any{
		"scopes_supported": []any{
			"openid",
			"profile",
			"workspace.todo:read",
			"workspace.todo:write",
			"workspace.todo:read",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"openid",
		"profile",
		"workspace.todo:read",
		"workspace.todo:write",
	}
	if len(scopes) != len(want) {
		t.Fatalf("scopes = %#v, want %#v", scopes, want)
	}
	for i := range want {
		if scopes[i] != want[i] {
			t.Fatalf("scopes = %#v, want %#v", scopes, want)
		}
	}
}

func TestOAuthScopesFromMetadataRejectsMissingOrInvalidScopes(t *testing.T) {
	tests := []struct {
		name string
		meta map[string]any
	}{
		{name: "missing", meta: map[string]any{}},
		{name: "wrong type", meta: map[string]any{"scopes_supported": "openid profile"}},
		{name: "empty", meta: map[string]any{"scopes_supported": []any{}}},
		{name: "invalid item", meta: map[string]any{"scopes_supported": []any{"openid", 42}}},
		{name: "blank item", meta: map[string]any{"scopes_supported": []any{"openid", " "}}},
		{name: "missing openid", meta: map[string]any{"scopes_supported": []any{"profile"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := oauthScopesFromMetadata(tt.meta); err == nil {
				t.Fatal("expected invalid metadata error")
			}
		})
	}
}

func TestOAuthCallbackHandlerDeliversOnlyFirstRequest(t *testing.T) {
	results := make(chan callbackResult, 1)
	handler := oauthCallbackHandler(results)

	first := httptest.NewRecorder()
	handler(first, httptest.NewRequest(http.MethodGet, "/callback?code=first&state=state-1", nil))
	if first.Code != http.StatusOK {
		t.Fatalf("first status = %d", first.Code)
	}
	result := <-results
	if result.code != "first" || result.state != "state-1" || result.err != "" {
		t.Fatalf("result = %#v", result)
	}

	repeated := httptest.NewRecorder()
	handler(repeated, httptest.NewRequest(http.MethodGet, "/callback?code=second&state=state-2", nil))
	if repeated.Code != http.StatusConflict {
		t.Fatalf("repeated status = %d, want %d", repeated.Code, http.StatusConflict)
	}
	select {
	case extra := <-results:
		t.Fatalf("unexpected repeated result: %#v", extra)
	default:
	}
}

func TestOAuthCallbackHandlerDoesNotBlockWhenResultBufferIsFull(t *testing.T) {
	results := make(chan callbackResult, 1)
	results <- callbackResult{code: "occupied"}
	handler := oauthCallbackHandler(results)
	done := make(chan struct{})
	response := httptest.NewRecorder()
	go func() {
		handler(response, httptest.NewRequest(http.MethodGet, "/callback?error=denied", nil))
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("callback handler blocked on a full result channel")
	}
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusServiceUnavailable)
	}
}

func TestCallbackRedirectURIMatchesLoopbackListener(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = listener.Close()
	})

	redirect, err := url.Parse(callbackRedirectURI(listener.Addr()))
	if err != nil {
		t.Fatal(err)
	}
	if redirect.Hostname() != "127.0.0.1" {
		t.Fatalf("redirect hostname = %q, want 127.0.0.1", redirect.Hostname())
	}
	if redirect.Port() != strconv.Itoa(listener.Addr().(*net.TCPAddr).Port) {
		t.Fatalf("redirect port = %q, listener = %q", redirect.Port(), listener.Addr())
	}
	if redirect.Path != "/callback" {
		t.Fatalf("redirect path = %q, want /callback", redirect.Path)
	}
}

func TestVerifiedTokenToCredentialUsesFallbacks(t *testing.T) {
	vt := &VerifiedToken{
		AccessToken: "access",
		ExpiresIn:   120,
	}
	cred, err := verifiedTokenToCredential("client", "https://example.test", vt, "refresh", "openid", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if cred.ClientID != "client" || cred.AccessToken != "access" || cred.RefreshToken != "refresh" || cred.Scope != "openid" {
		t.Fatalf("credential = %#v", cred)
	}
	if cred.ExpiresAt == 0 {
		t.Fatal("ExpiresAt was not populated")
	}
	if cred.Resource != "https://example.test" {
		t.Fatalf("resource = %q, want %q", cred.Resource, "https://example.test")
	}
}

func TestVerifiedTokenToCredentialRequiresAccessToken(t *testing.T) {
	vt := &VerifiedToken{}
	if _, err := verifiedTokenToCredential("client", "resource", vt, "", "", time.Now()); err == nil {
		t.Fatal("expected missing access token error")
	}
}

func TestVerifiedTokenToCredentialNilGuard(t *testing.T) {
	if _, err := verifiedTokenToCredential("client", "resource", nil, "", "", time.Now()); err == nil {
		t.Fatal("expected error for nil token")
	}
}

func TestRequireIDTokenForOpenID(t *testing.T) {
	if err := requireIDTokenForOpenID("openid profile", ""); err == nil {
		t.Fatal("expected error when openid scope requested without id_token")
	}
	if err := requireIDTokenForOpenID("profile email", ""); err != nil {
		t.Fatalf("unexpected error when openid not requested: %v", err)
	}
	if err := requireIDTokenForOpenID("openid", "idtoken"); err != nil {
		t.Fatalf("unexpected error when id_token present: %v", err)
	}
}

func TestEffectiveTokenScopePrefersGrantedScope(t *testing.T) {
	vt := &VerifiedToken{Scope: "profile workspace.todo:read"}
	if got := effectiveTokenScope(vt, "openid profile workspace.todo:read"); got != "profile workspace.todo:read" {
		t.Fatalf("effective scope = %q", got)
	}
	if err := requireIDTokenForOpenID(effectiveTokenScope(vt, "openid profile"), ""); err != nil {
		t.Fatalf("reduced grant without openid should not require an ID token: %v", err)
	}
}

func TestRefreshTokenDoesNotRequireNewIDToken(t *testing.T) {
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/.well-known/oauth-authorization-server/api/auth":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"issuer":         server.URL + "/api/auth",
				"token_endpoint": server.URL + "/api/auth/oauth2/token",
			})
		case "/api/auth/oauth2/token":
			if err := r.ParseForm(); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if r.Form.Get("grant_type") != "refresh_token" {
				http.Error(w, "unexpected grant", http.StatusBadRequest)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "next-access",
				"token_type":   "Bearer",
				"expires_in":   3600,
			})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	cred, err := RefreshToken(server.URL, &config.Credential{
		ClientID:     "client-1",
		RefreshToken: "refresh-1",
		Scope:        "openid profile workspace.todo:read",
	})
	if err != nil {
		t.Fatal(err)
	}
	if cred.AccessToken != "next-access" || cred.RefreshToken != "refresh-1" {
		t.Fatalf("credential = %#v", cred)
	}
	if cred.Scope != "openid profile workspace.todo:read" {
		t.Fatalf("scope = %q", cred.Scope)
	}
}

func TestValidateIDTokenAudienceIsClientID(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: key}, nil)
	if err != nil {
		t.Fatal(err)
	}
	build := func(aud any) string {
		t.Helper()
		claims := map[string]any{
			"iss": "https://issuer.test",
			"aud": aud,
			"exp": time.Now().Add(time.Hour).Unix(),
		}
		s, err := jwt.Signed(signer).Claims(claims).Serialize()
		if err != nil {
			t.Fatal(err)
		}
		return s
	}

	vt := &VerifiedToken{IDToken: build("client-id-123")}
	if err := vt.ValidateIDToken("https://issuer.test", "client-id-123"); err != nil {
		t.Fatalf("expected client_id audience to validate: %v", err)
	}

	vt.IDToken = build("https://server.test")
	if err := vt.ValidateIDToken("https://issuer.test", "client-id-123"); err == nil {
		t.Fatal("expected server URL audience to fail against client_id expectation")
	}

	vt.IDToken = build([]string{"client-id-123", "other"})
	if err := vt.ValidateIDToken("https://issuer.test", "client-id-123"); err != nil {
		t.Fatalf("expected audience list containing client_id to validate: %v", err)
	}
}

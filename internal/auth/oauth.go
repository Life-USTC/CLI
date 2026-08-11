package auth

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Life-USTC/CLI/internal/config"
	"golang.org/x/oauth2"
)

type oauthToken struct {
	AccessToken  string
	RefreshToken string
	TokenType    string
	Expiry       time.Time
	ExpiresIn    int
	Scope        string
}

func newOAuthToken(tok *oauth2.Token) *oauthToken {
	if tok == nil {
		return nil
	}
	return &oauthToken{
		AccessToken:  tok.AccessToken,
		RefreshToken: tok.RefreshToken,
		TokenType:    tok.TokenType,
		Expiry:       tok.Expiry,
		ExpiresIn:    tokenExpiresIn(tok, 0),
		Scope:        tokenExtraString(tok, "scope"),
	}
}

func tokenExtraString(tok *oauth2.Token, key string) string {
	if tok == nil {
		return ""
	}
	if s, ok := tok.Extra(key).(string); ok {
		return strings.TrimSpace(s)
	}
	return ""
}

func parseIntString(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, errors.New("empty")
	}
	return strconv.ParseInt(s, 10, 64)
}

func effectiveTokenScope(token *oauthToken, fallback string) string {
	if token != nil && strings.TrimSpace(token.Scope) != "" {
		return strings.TrimSpace(token.Scope)
	}
	return strings.TrimSpace(fallback)
}

func oauthTokenToCredential(clientID, resource string, token *oauthToken, fallbackRefresh, fallbackScope string, now time.Time) (*config.Credential, error) {
	if token == nil {
		return nil, errors.New("token response is nil")
	}
	accessToken := strings.TrimSpace(token.AccessToken)
	if accessToken == "" {
		return nil, errors.New("token response missing access_token")
	}
	expiresIn := token.ExpiresIn
	if expiresIn <= 0 {
		expiresIn = 3600
	}
	refreshToken := strings.TrimSpace(token.RefreshToken)
	if refreshToken == "" {
		refreshToken = fallbackRefresh
	}
	scope := effectiveTokenScope(token, fallbackScope)
	return &config.Credential{
		ClientID:     clientID,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    strings.TrimSpace(token.TokenType),
		ExpiresAt:    float64(now.Add(time.Duration(expiresIn) * time.Second).Unix()),
		Scope:        scope,
		Resource:     resource,
	}, nil
}

func oauth2Context(ctx context.Context, client *http.Client) context.Context {
	return context.WithValue(ctx, oauth2.HTTPClient, client)
}

func stringFromMap(m map[string]any, key string) string {
	if s, ok := m[key].(string); ok {
		return strings.TrimSpace(s)
	}
	return ""
}

func tokenExpiresIn(tok *oauth2.Token, fallback int) int {
	if tok == nil {
		return fallback
	}
	if extra := tok.Extra("expires_in"); extra != nil {
		switch v := extra.(type) {
		case int:
			return v
		case int64:
			return int(v)
		case float64:
			if v > 0 {
				return int(v)
			}
		case string:
			if n, err := parseIntString(v); err == nil && n > 0 {
				return int(n)
			}
		}
	}
	if !tok.Expiry.IsZero() {
		secs := int(time.Until(tok.Expiry).Seconds())
		if secs > 0 {
			return secs
		}
	}
	return fallback
}

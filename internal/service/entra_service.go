package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"

	"ulas-service/internal/config"
)

var ErrEntraNotConfigured = errors.New("entra id login is not configured")

type EntraIdentity struct {
	Username string
	Name     string
	Email    string
	Subject  string // immutable Entra object ID (oid claim) — kept for future RBAC
}

// EntraService performs the Entra ID (OIDC) authorization-code flow.
type EntraService interface {
	// Configured reports whether Entra ID login is enabled.
	Configured() bool
	// AuthURL returns the Entra authorize URL to redirect the browser to.
	// The state value must be verified against the value sent in the callback.
	AuthURL(ctx context.Context, state string) (string, error)
	// Exchange trades an authorization code for a verified EntraIdentity.
	Exchange(ctx context.Context, code string) (*EntraIdentity, error)
}

type DefaultEntraService struct {
	cfg *config.AppConfig

	mu       sync.Mutex
	provider *oidc.Provider
	oauth2   *oauth2.Config
}

func NewEntraService(cfg *config.AppConfig) *DefaultEntraService {
	if cfg == nil || cfg.EntraTenantID == "" || cfg.EntraClientID == "" || cfg.EntraClientSecret == "" || cfg.EntraRedirectURL == "" {
		return nil
	}
	return &DefaultEntraService{cfg: cfg}
}

func (s *DefaultEntraService) Configured() bool { return s != nil }

// init lazily discovers the OIDC provider and builds the oauth2 config.
func (s *DefaultEntraService) init(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.oauth2 != nil {
		return nil
	}
	provider, err := oidc.NewProvider(ctx, "https://login.microsoftonline.com/"+s.cfg.EntraTenantID+"/v2.0")
	if err != nil {
		return fmt.Errorf("entra oidc discovery: %w", err)
	}
	s.provider = provider
	s.oauth2 = &oauth2.Config{
		ClientID:     s.cfg.EntraClientID,
		ClientSecret: s.cfg.EntraClientSecret,
		Endpoint:     provider.Endpoint(),
		RedirectURL:  s.cfg.EntraRedirectURL,
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
	}
	return nil
}

func (s *DefaultEntraService) AuthURL(ctx context.Context, state string) (string, error) {
	if !s.Configured() {
		return "", ErrEntraNotConfigured
	}
	if err := s.init(ctx); err != nil {
		return "", err
	}
	return s.oauth2.AuthCodeURL(state), nil
}

func (s *DefaultEntraService) Exchange(ctx context.Context, code string) (*EntraIdentity, error) {
	if !s.Configured() {
		return nil, ErrEntraNotConfigured
	}
	if err := s.init(ctx); err != nil {
		return nil, err
	}
	token, err := s.oauth2.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("entra token exchange: %w", err)
	}
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		return nil, errors.New("entra response carried no id_token")
	}
	verifier := s.provider.Verifier(&oidc.Config{ClientID: s.cfg.EntraClientID})
	idToken, err := verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return nil, fmt.Errorf("entra id token verification: %w", err)
	}
	var claims struct {
		PreferredUsername string `json:"preferred_username"`
		Email             string `json:"email"`
		Name              string `json:"name"`
		OID               string `json:"oid"`
	}
	if err := idToken.Claims(&claims); err != nil {
		return nil, fmt.Errorf("entra id token claims: %w", err)
	}

	username := normalizeEntraUsername(claims.PreferredUsername, claims.Email, idToken.Subject)
	if username == "" {
		return nil, errors.New("entra token carried no usable username claim")
	}
	return &EntraIdentity{
		Username: username,
		Name:     claims.Name,
		Email:    claims.Email,
		Subject:  claims.OID,
	}, nil
}

// normalizeEntraUsername derives the canonical username from Entra claims:
// lowercased UPN/email prefix without the domain, matching the comsiid format
// used by LDAP logins and the user_access table.
func normalizeEntraUsername(preferredUsername, email, fallback string) string {
	upn := preferredUsername
	if upn == "" {
		upn = email
	}
	if upn == "" {
		upn = fallback
	}
	if at := strings.Index(upn, "@"); at >= 0 {
		upn = upn[:at]
	}
	return strings.ToLower(strings.TrimSpace(upn))
}

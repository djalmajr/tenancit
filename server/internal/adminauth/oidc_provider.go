package adminauth

import (
	"context"
	"errors"
	"fmt"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

type Provider struct {
	oauthConfig oauth2.Config
	verifier    *oidc.IDTokenVerifier
	issuer      string
	roleClaim   string
}

func NewProvider(ctx context.Context, config OIDCConfig) (*Provider, error) {
	provider, err := oidc.NewProvider(ctx, config.Issuer)
	if err != nil {
		return nil, fmt.Errorf("OIDC discovery: %w", err)
	}
	return &Provider{
		oauthConfig: oauth2.Config{
			ClientID: config.ClientID, ClientSecret: config.ClientSecret,
			Endpoint: provider.Endpoint(), RedirectURL: config.RedirectURL,
			Scopes: []string{oidc.ScopeOpenID, "profile", "email"},
		},
		verifier: provider.Verifier(&oidc.Config{ClientID: config.ClientID}),
		issuer:   config.Issuer, roleClaim: config.RoleClaim,
	}, nil
}

func (p *Provider) AuthorizationURL(state, nonce, challenge string) string {
	return p.oauthConfig.AuthCodeURL(
		state,
		oauth2.AccessTypeOffline,
		oauth2.SetAuthURLParam("nonce", nonce),
		oauth2.SetAuthURLParam("code_challenge", challenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	)
}

func (p *Provider) Exchange(ctx context.Context, code, verifier string) (OIDCClaims, error) {
	token, err := p.oauthConfig.Exchange(ctx, code, oauth2.SetAuthURLParam("code_verifier", verifier))
	if err != nil {
		return OIDCClaims{}, fmt.Errorf("OIDC code exchange: %w", err)
	}
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok || rawIDToken == "" {
		return OIDCClaims{}, errors.New("OIDC response omitted id_token")
	}
	idToken, err := p.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return OIDCClaims{}, fmt.Errorf("OIDC id_token verification: %w", err)
	}
	var rawClaims map[string]any
	if err := idToken.Claims(&rawClaims); err != nil {
		return OIDCClaims{}, fmt.Errorf("OIDC claims: %w", err)
	}
	return claimsFromMap(rawClaims, p.issuer, p.roleClaim)
}

func claimsFromMap(raw map[string]any, expectedIssuer, roleClaim string) (OIDCClaims, error) {
	issuer, _ := raw["iss"].(string)
	subject, _ := raw["sub"].(string)
	nonce, _ := raw["nonce"].(string)
	label, _ := raw["name"].(string)
	if label == "" {
		label, _ = raw["preferred_username"].(string)
	}
	if label == "" {
		label, _ = raw["email"].(string)
	}
	if issuer == "" {
		issuer = expectedIssuer
	}
	if subject == "" || nonce == "" {
		return OIDCClaims{}, errors.New("OIDC id_token is missing sub or nonce")
	}
	roleValues, err := stringClaimValues(raw[roleClaim])
	if err != nil {
		return OIDCClaims{}, fmt.Errorf("OIDC role claim %q: %w", roleClaim, err)
	}
	return OIDCClaims{Issuer: issuer, Subject: subject, Label: label, Nonce: nonce, RoleValues: roleValues}, nil
}

func stringClaimValues(value any) ([]string, error) {
	switch typed := value.(type) {
	case string:
		if typed == "" {
			return nil, errors.New("empty value")
		}
		return []string{typed}, nil
	case []any:
		values := make([]string, 0, len(typed))
		for _, item := range typed {
			text, ok := item.(string)
			if !ok || text == "" {
				return nil, errors.New("must contain only non-empty strings")
			}
			values = append(values, text)
		}
		if len(values) == 0 {
			return nil, errors.New("must not be empty")
		}
		return values, nil
	default:
		return nil, errors.New("must be a string or string array")
	}
}

package adminauth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"io"
	"net/url"
	"strings"
	"time"

	appcrypto "github.com/djalmajr/tenancit/server/internal/crypto"
)

const loginAttemptTTL = 10 * time.Minute

type OIDCClaims struct {
	Issuer     string
	Subject    string
	Label      string
	Nonce      string
	RoleValues []string
}

type oidcProvider interface {
	AuthorizationURL(state, nonce, challenge string) string
	Exchange(context.Context, string, string) (OIDCClaims, error)
}

type CreateLoginAttemptParams struct {
	StateHash, NonceHash            string
	PKCEVerifierCipher, CipherNonce []byte
	KeyVersion                      int16
	RedirectAfter                   string
	CreatedAt, ExpiresAt            time.Time
}

type StoredLoginAttempt struct {
	StateHash, NonceHash            string
	PKCEVerifierCipher, CipherNonce []byte
	KeyVersion                      int16
	RedirectAfter                   string
	CreatedAt, ExpiresAt            time.Time
}

type loginAttemptStore interface {
	CreateLoginAttempt(context.Context, CreateLoginAttemptParams) (StoredLoginAttempt, error)
	ConsumeLoginAttempt(context.Context, string, time.Time) (StoredLoginAttempt, error)
}

type LoginStart struct {
	AuthorizationURL string
}

type CompletedLogin struct {
	CreatedSession
	RedirectAfter string
}

type oidcFlowError struct {
	stage string
	err   error
}

func (e *oidcFlowError) Error() string { return "OIDC flow failed at " + e.stage }
func (e *oidcFlowError) Unwrap() error { return e.err }

func flowFailure(stage string, err error) error {
	return &oidcFlowError{stage: stage, err: err}
}

func FailureStage(err error) string {
	var flowError *oidcFlowError
	if errors.As(err, &flowError) {
		return flowError.stage
	}
	return "unknown"
}

type OIDCManager struct {
	config   OIDCConfig
	provider oidcProvider
	attempts loginAttemptStore
	sessions *SessionManager
	cryptor  *appcrypto.Cryptor
	random   io.Reader
	now      func() time.Time
}

func NewOIDCManager(config OIDCConfig, provider oidcProvider, attempts loginAttemptStore, sessions *SessionManager, cryptor *appcrypto.Cryptor, random io.Reader, now func() time.Time) *OIDCManager {
	if random == nil {
		random = rand.Reader
	}
	if now == nil {
		now = time.Now
	}
	return &OIDCManager{config: config, provider: provider, attempts: attempts, sessions: sessions, cryptor: cryptor, random: random, now: now}
}

func (m *OIDCManager) Start(ctx context.Context, redirectAfter string) (LoginStart, error) {
	if m.provider == nil || m.attempts == nil || m.sessions == nil || m.cryptor == nil || m.random == nil {
		return LoginStart{}, errors.New("OIDC manager is not fully configured")
	}
	redirectAfter, err := safeRedirectAfter(redirectAfter, m.config.BasePath)
	if err != nil {
		return LoginStart{}, err
	}
	state, err := randomCredential(m.random)
	if err != nil {
		return LoginStart{}, err
	}
	nonce, err := randomCredential(m.random)
	if err != nil {
		return LoginStart{}, err
	}
	verifier, err := randomCredential(m.random)
	if err != nil {
		return LoginStart{}, err
	}
	encrypted, err := m.cryptor.Encrypt(verifier)
	if err != nil {
		return LoginStart{}, err
	}
	now := m.now().UTC()
	_, err = m.attempts.CreateLoginAttempt(ctx, CreateLoginAttemptParams{
		StateHash: HashCredential(state), NonceHash: HashCredential(nonce),
		PKCEVerifierCipher: encrypted.Cipher, CipherNonce: encrypted.Nonce,
		KeyVersion: int16(encrypted.KeyVersion), RedirectAfter: redirectAfter,
		CreatedAt: now, ExpiresAt: now.Add(loginAttemptTTL),
	})
	if err != nil {
		return LoginStart{}, err
	}
	return LoginStart{AuthorizationURL: m.provider.AuthorizationURL(state, nonce, pkceChallenge(verifier))}, nil
}

func (m *OIDCManager) Complete(ctx context.Context, state, code string) (CompletedLogin, error) {
	state = strings.TrimSpace(state)
	code = strings.TrimSpace(code)
	if state == "" || code == "" || m.attempts == nil || m.provider == nil || m.sessions == nil || m.cryptor == nil {
		return CompletedLogin{}, errors.New("invalid OIDC callback")
	}
	attempt, err := m.attempts.ConsumeLoginAttempt(ctx, HashCredential(state), m.now().UTC())
	if err != nil {
		return CompletedLogin{}, flowFailure("state", err)
	}
	verifier, err := m.cryptor.Decrypt(appcrypto.Encrypted{
		Cipher: attempt.PKCEVerifierCipher, Nonce: attempt.CipherNonce, KeyVersion: int(attempt.KeyVersion),
	})
	if err != nil {
		return CompletedLogin{}, flowFailure("pkce", err)
	}
	claims, err := m.provider.Exchange(ctx, code, verifier)
	if err != nil {
		return CompletedLogin{}, flowFailure("exchange", err)
	}
	if err := validateOIDCIdentity(m.config.Issuer, claims.Issuer, claims.Subject); err != nil {
		return CompletedLogin{}, flowFailure("identity", err)
	}
	if err := validateNonce(attempt.NonceHash, claims.Nonce); err != nil {
		return CompletedLogin{}, flowFailure("nonce", err)
	}
	roles, err := mapRoles(claims.RoleValues, m.config.RoleMappings)
	if err != nil {
		return CompletedLogin{}, flowFailure("roles", err)
	}
	created, err := m.sessions.Create(ctx, SessionIdentity{
		Issuer: claims.Issuer, Subject: claims.Subject, Label: claims.Label,
		Roles: roles, Permissions: permissionsForRoles(roles),
	})
	if err != nil {
		return CompletedLogin{}, flowFailure("session", err)
	}
	return CompletedLogin{CreatedSession: created, RedirectAfter: attempt.RedirectAfter}, nil
}

func validateOIDCIdentity(expectedIssuer, actualIssuer, subject string) error {
	if strings.TrimSuffix(strings.TrimSpace(actualIssuer), "/") != strings.TrimSuffix(strings.TrimSpace(expectedIssuer), "/") || strings.TrimSpace(subject) == "" {
		return errors.New("OIDC identity does not match configured issuer")
	}
	return nil
}

func pkceChallenge(verifier string) string {
	digest := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(digest[:])
}

func validateNonce(expectedHash, actual string) error {
	actualHash := HashCredential(actual)
	if actual == "" || subtle.ConstantTimeCompare([]byte(expectedHash), []byte(actualHash)) != 1 {
		return errors.New("OIDC nonce mismatch")
	}
	return nil
}

func mapRoles(values []string, mappings map[string]Role) ([]Role, error) {
	seen := map[Role]struct{}{}
	roles := make([]Role, 0, len(values))
	for _, value := range values {
		role, ok := mappings[value]
		if !ok {
			continue
		}
		if _, duplicate := seen[role]; duplicate {
			continue
		}
		seen[role] = struct{}{}
		roles = append(roles, role)
	}
	if len(roles) == 0 {
		return nil, errors.New("OIDC identity has no mapped role")
	}
	return roles, nil
}

func permissionsForRoles(roles []Role) []string {
	byRole := map[Role][]string{
		RoleViewer:        {"admin.read"},
		RoleOperator:      {"admin.read", "tenant.write", "resource.write", "api_client.manage", "integration.manage"},
		RoleSecurityAdmin: {"admin.read", "audit.read", "audit.export", "audit.manage", "api_client.manage", "integration.manage", "resource.write", "secret.reveal", "session.manage", "settings.manage", "tenant.hard_delete", "tenant.write"},
	}
	seen := map[string]struct{}{}
	permissions := []string{}
	for _, role := range roles {
		for _, permission := range byRole[role] {
			if _, duplicate := seen[permission]; duplicate {
				continue
			}
			seen[permission] = struct{}{}
			permissions = append(permissions, permission)
		}
	}
	return permissions
}

func safeRedirectAfter(value, rawBasePath string) (string, error) {
	basePath, err := normalizeBasePath(rawBasePath)
	if err != nil {
		return "", err
	}
	value = strings.TrimSpace(value)
	if value == "" {
		if basePath == "/" {
			return "/", nil
		}
		return basePath + "/", nil
	}
	if !strings.HasPrefix(value, "/") || strings.HasPrefix(value, "//") || strings.ContainsAny(value, "\\\r\n") {
		return "", errors.New("invalid post-login redirect")
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.IsAbs() || parsed.Host != "" || parsed.Opaque != "" {
		return "", errors.New("invalid post-login redirect")
	}
	decodedPath := parsed.Path
	if decodedPath == "" || strings.ContainsAny(decodedPath, "\\\r\n") {
		return "", errors.New("invalid post-login redirect")
	}
	for _, segment := range strings.Split(decodedPath, "/") {
		if segment == "." || segment == ".." {
			return "", errors.New("invalid post-login redirect")
		}
	}
	if basePath != "/" && decodedPath != basePath && !strings.HasPrefix(decodedPath, basePath+"/") {
		return "", errors.New("post-login redirect escapes configured base path")
	}
	return value, nil
}

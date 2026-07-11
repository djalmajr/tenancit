package adminauth

import (
	"bytes"
	"context"
	"slices"
	"testing"
	"time"

	appcrypto "github.com/djalmajr/tenancit/server/internal/crypto"
)

type fakeAttemptStore struct {
	created CreateLoginAttemptParams
	record  StoredLoginAttempt
}

func (f *fakeAttemptStore) CreateLoginAttempt(_ context.Context, params CreateLoginAttemptParams) (StoredLoginAttempt, error) {
	f.created = params
	f.record = StoredLoginAttempt(params)
	return f.record, nil
}

func (f *fakeAttemptStore) ConsumeLoginAttempt(_ context.Context, stateHash string, _ time.Time) (StoredLoginAttempt, error) {
	if stateHash != f.record.StateHash {
		return StoredLoginAttempt{}, mismatchError{}
	}
	return f.record, nil
}

type mismatchError struct{}

func (mismatchError) Error() string { return "state mismatch" }

type fakeProvider struct {
	state, nonce, challenge string
	code, verifier          string
	claims                  OIDCClaims
}

func (f *fakeProvider) AuthorizationURL(state, nonce, challenge string) string {
	f.state, f.nonce, f.challenge = state, nonce, challenge
	return "https://id.example.test/authorize?state=" + state
}

func (f *fakeProvider) Exchange(_ context.Context, code, verifier string) (OIDCClaims, error) {
	f.code, f.verifier = code, verifier
	claims := f.claims
	claims.Nonce = f.nonce
	return claims, nil
}

func TestOIDCManagerPersistsHashedStateAndCompletesOneTimeSession(t *testing.T) {
	key := bytes.Repeat([]byte{0x55}, 32)
	cryptor, err := appcrypto.New(map[int][]byte{1: key}, 1)
	if err != nil {
		t.Fatalf("crypto.New: %v", err)
	}
	attempts := &fakeAttemptStore{}
	provider := &fakeProvider{claims: OIDCClaims{
		Issuer: "https://id.example.test", Subject: "user-1", Label: "Ada",
		RoleValues: []string{"platform-operators"},
	}}
	sessions := &fakeSessionStore{}
	now := time.Date(2026, 7, 11, 12, 0, 0, 0, time.UTC)
	randomBytes := make([]byte, 256)
	for index := range randomBytes {
		randomBytes[index] = byte(index)
	}
	sessionManager := NewSessionManager(sessions, cryptor, bytes.NewReader(randomBytes[128:]), func() time.Time { return now }, 8*time.Hour, 30*time.Minute)
	manager := NewOIDCManager(OIDCConfig{
		Issuer: "https://id.example.test", RoleMappings: map[string]Role{"platform-operators": RoleOperator},
	}, provider, attempts, sessionManager, cryptor, bytes.NewReader(randomBytes[:128]), func() time.Time { return now })

	start, err := manager.Start(context.Background(), "/tenants")
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if start.AuthorizationURL == "" || provider.state == "" || provider.nonce == "" || provider.challenge == "" {
		t.Fatalf("start/provider=%+v/%+v", start, provider)
	}
	if attempts.created.StateHash == provider.state || attempts.created.NonceHash == provider.nonce {
		t.Fatal("raw OIDC state or nonce reached persistence")
	}
	if attempts.created.PKCEVerifierCipher == nil || bytes.Contains(attempts.created.PKCEVerifierCipher, []byte(provider.challenge)) {
		t.Fatal("PKCE verifier was not encrypted independently of its challenge")
	}

	created, err := manager.Complete(context.Background(), provider.state, "authorization-code")
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if provider.code != "authorization-code" || provider.verifier == "" {
		t.Fatalf("exchange code=%q verifier=%q", provider.code, provider.verifier)
	}
	if created.Identity.Issuer != provider.claims.Issuer || created.Identity.Subject != provider.claims.Subject {
		t.Fatalf("identity=%+v", created.Identity)
	}
	if len(created.Identity.Roles) != 1 || created.Identity.Roles[0] != RoleOperator {
		t.Fatalf("roles=%v", created.Identity.Roles)
	}
	if created.Token == "" || created.CSRFToken == "" {
		t.Fatal("OIDC completion did not create a session")
	}
}

func TestOIDCManagerRejectsUnmappedRoleAndNonceMismatch(t *testing.T) {
	// These validations are fail-closed before a session can be persisted.
	if _, err := mapRoles([]string{"unknown"}, map[string]Role{"admins": RoleSecurityAdmin}); err == nil {
		t.Fatal("unmapped role accepted")
	}
	if err := validateNonce(HashCredential("expected"), "other"); err == nil {
		t.Fatal("nonce mismatch accepted")
	}
	if err := validateOIDCIdentity("https://id.example.test", "https://spoofed.example.test", "user-1"); err == nil {
		t.Fatal("spoofed issuer accepted")
	}
}

func TestPermissionsForRolesMatchGovernedCapabilities(t *testing.T) {
	operator := permissionsForRoles([]Role{RoleOperator})
	if !slices.Contains(operator, "integration.manage") || slices.Contains(operator, "settings.manage") {
		t.Fatalf("operator permissions=%v", operator)
	}
	securityAdmin := permissionsForRoles([]Role{RoleSecurityAdmin})
	for _, permission := range []string{"audit.manage", "integration.manage", "session.manage", "settings.manage", "tenant.hard_delete"} {
		if !slices.Contains(securityAdmin, permission) {
			t.Fatalf("security admin missing %q: %v", permission, securityAdmin)
		}
	}
}

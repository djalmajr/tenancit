package adminauth

import (
	"bytes"
	"context"
	"testing"
	"time"

	appcrypto "github.com/djalmajr/tenancit/server/internal/crypto"
	"github.com/google/uuid"
)

type fakeSessionStore struct {
	created     CreateSessionParams
	auth        AuthenticateSessionParams
	session     StoredSession
	createError error
	authError   error
}

func (f *fakeSessionStore) CreateAdminSession(_ context.Context, params CreateSessionParams) (StoredSession, error) {
	f.created = params
	if f.createError != nil {
		return StoredSession{}, f.createError
	}
	f.session = StoredSession{
		ID: paramsToUUID(), TokenHash: params.TokenHash, CSRFTokenHash: params.CSRFTokenHash,
		CSRFTokenCipher: params.CSRFTokenCipher, CSRFNonce: params.CSRFNonce, CSRFKeyVersion: params.CSRFKeyVersion,
		ActorIssuer: params.ActorIssuer, ActorSubject: params.ActorSubject, ActorLabel: params.ActorLabel,
		Roles: params.Roles, Permissions: params.Permissions, CreatedAt: params.CreatedAt,
		LastUsedAt: params.CreatedAt, ExpiresAt: params.ExpiresAt, IdleExpiresAt: params.IdleExpiresAt,
	}
	return f.session, nil
}

func (f *fakeSessionStore) AuthenticateAdminSession(_ context.Context, params AuthenticateSessionParams) (StoredSession, error) {
	f.auth = params
	return f.session, f.authError
}

func (f *fakeSessionStore) RevokeAdminSession(context.Context, uuid.UUID, time.Time) error {
	return nil
}

func TestSessionManagerCreatesHashOnlyCredentials(t *testing.T) {
	store := &fakeSessionStore{}
	now := time.Date(2026, 7, 11, 12, 0, 0, 0, time.UTC)
	randomBytes := make([]byte, 128)
	for index := range randomBytes {
		randomBytes[index] = byte(index)
	}
	manager := NewSessionManager(store, testCryptor(t), bytes.NewReader(randomBytes), func() time.Time { return now }, 8*time.Hour, 30*time.Minute)

	created, err := manager.Create(context.Background(), SessionIdentity{
		Issuer: "https://id.example.test", Subject: "user-1", Label: "Ada",
		Roles: []Role{RoleOperator}, Permissions: []string{"admin.read", "tenant.write"},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.Token == "" || created.CSRFToken == "" || created.Token == created.CSRFToken {
		t.Fatal("session credentials were not independently generated")
	}
	if store.created.TokenHash == created.Token || store.created.CSRFTokenHash == created.CSRFToken {
		t.Fatal("raw reusable credential reached persistence")
	}
	if store.created.TokenHash != HashCredential(created.Token) || store.created.CSRFTokenHash != HashCredential(created.CSRFToken) {
		t.Fatal("persisted credential hashes do not match returned one-shot values")
	}
	if !store.created.ExpiresAt.Equal(now.Add(8*time.Hour)) || !store.created.IdleExpiresAt.Equal(now.Add(30*time.Minute)) {
		t.Fatalf("expiry absolute=%v idle=%v", store.created.ExpiresAt, store.created.IdleExpiresAt)
	}
}

func TestSessionManagerAuthenticatesByHashAndValidatesCSRF(t *testing.T) {
	store := &fakeSessionStore{session: StoredSession{
		ID: paramsToUUID(), ActorIssuer: "https://id.example.test", ActorSubject: "user-1",
		Roles: []string{"operator"}, Permissions: []string{"admin.read"}, CSRFTokenHash: HashCredential("csrf-token"),
	}}
	encrypted, err := testCryptor(t).Encrypt("csrf-token")
	if err != nil {
		t.Fatalf("encrypt CSRF fixture: %v", err)
	}
	store.session.CSRFTokenCipher, store.session.CSRFNonce, store.session.CSRFKeyVersion = encrypted.Cipher, encrypted.Nonce, int16(encrypted.KeyVersion)
	now := time.Date(2026, 7, 11, 12, 0, 0, 0, time.UTC)
	manager := NewSessionManager(store, testCryptor(t), bytes.NewReader(bytes.Repeat([]byte{0x33}, 128)), func() time.Time { return now }, 8*time.Hour, 30*time.Minute)

	authenticated, err := manager.Authenticate(context.Background(), "raw-session-token")
	if err != nil {
		t.Fatalf("Authenticate: %v", err)
	}
	if store.auth.TokenHash != HashCredential("raw-session-token") || store.auth.IdleSeconds != 1800 {
		t.Fatalf("auth params=%+v", store.auth)
	}
	if authenticated.Subject != "user-1" || authenticated.SessionID == "" {
		t.Fatalf("identity=%+v", authenticated)
	}
	if !ValidateCSRF(store.session.CSRFTokenHash, "csrf-token") {
		t.Fatal("valid CSRF token rejected")
	}
	if ValidateCSRF(store.session.CSRFTokenHash, "wrong") {
		t.Fatal("invalid CSRF token accepted")
	}
}

func paramsToUUID() uuid.UUID {
	return uuid.MustParse("11111111-1111-4111-8111-111111111111")
}

func testCryptor(t *testing.T) *appcrypto.Cryptor {
	t.Helper()
	cryptor, err := appcrypto.New(map[int][]byte{1: bytes.Repeat([]byte{0x77}, 32)}, 1)
	if err != nil {
		t.Fatalf("crypto.New: %v", err)
	}
	return cryptor
}

package adminauth_test

import (
	"context"
	"testing"
	"time"

	"github.com/djalmajr/tenancit/server/internal/adminauth"
	"github.com/djalmajr/tenancit/server/internal/testsupport"
	"github.com/jackc/pgx/v5"
)

func TestPostgresSessionStoreAuthenticatesAndExtendsIdleExpiry(t *testing.T) {
	pool := testsupport.NewDB(t)
	store := adminauth.NewPostgresSessionStore(pool)
	ctx := context.Background()
	createdAt := time.Date(2026, 7, 11, 12, 0, 0, 0, time.UTC)

	created, err := store.CreateAdminSession(ctx, adminauth.CreateSessionParams{
		TokenHash: "hash-only", CSRFTokenHash: "csrf-hash",
		CSRFTokenCipher: []byte{1, 2}, CSRFNonce: []byte{3, 4}, CSRFKeyVersion: 1,
		ActorIssuer: "https://id.example.test", ActorSubject: "user-1",
		Roles: []string{"viewer"}, Permissions: []string{"admin.read"},
		CreatedAt: createdAt, ExpiresAt: createdAt.Add(8 * time.Hour),
		IdleExpiresAt: createdAt.Add(30 * time.Minute),
	})
	if err != nil {
		t.Fatalf("CreateAdminSession: %v", err)
	}
	usedAt := createdAt.Add(10 * time.Minute)
	authenticated, err := store.AuthenticateAdminSession(ctx, adminauth.AuthenticateSessionParams{
		TokenHash: "hash-only", UsedAt: usedAt, IdleSeconds: 1800,
	})
	if err != nil {
		t.Fatalf("AuthenticateAdminSession: %v", err)
	}
	if authenticated.ID != created.ID || !authenticated.LastUsedAt.Equal(usedAt) {
		t.Fatalf("authenticated=%+v", authenticated)
	}
	if !authenticated.IdleExpiresAt.Equal(usedAt.Add(30 * time.Minute)) {
		t.Fatalf("idle expiry=%v", authenticated.IdleExpiresAt)
	}

	if _, err := store.AuthenticateAdminSession(ctx, adminauth.AuthenticateSessionParams{
		TokenHash: "raw-session-token", UsedAt: usedAt, IdleSeconds: 1800,
	}); err != pgx.ErrNoRows {
		t.Fatalf("raw token lookup err=%v want pgx.ErrNoRows", err)
	}
}

func TestPostgresLoginAttemptIsSingleUseAndExpiresAtBoundary(t *testing.T) {
	pool := testsupport.NewDB(t)
	store := adminauth.NewPostgresSessionStore(pool)
	ctx := context.Background()
	now := time.Date(2026, 7, 11, 12, 0, 0, 0, time.UTC)
	create := func(state string, expiresAt time.Time) {
		t.Helper()
		_, err := store.CreateLoginAttempt(ctx, adminauth.CreateLoginAttemptParams{
			StateHash: state, NonceHash: "nonce-" + state,
			PKCEVerifierCipher: []byte{1, 2}, CipherNonce: []byte{3, 4}, KeyVersion: 1,
			RedirectAfter: "/", CreatedAt: now.Add(-time.Minute), ExpiresAt: expiresAt,
		})
		if err != nil {
			t.Fatalf("create attempt: %v", err)
		}
	}

	create("one-use", now.Add(time.Second))
	if _, err := store.ConsumeLoginAttempt(ctx, "one-use", now); err != nil {
		t.Fatalf("first consume: %v", err)
	}
	if _, err := store.ConsumeLoginAttempt(ctx, "one-use", now); err != pgx.ErrNoRows {
		t.Fatalf("replay err=%v want pgx.ErrNoRows", err)
	}

	create("boundary", now)
	if _, err := store.ConsumeLoginAttempt(ctx, "boundary", now); err != pgx.ErrNoRows {
		t.Fatalf("exact expiry err=%v want pgx.ErrNoRows", err)
	}
}

func TestPostgresSessionRejectsAbsoluteAndIdleExpiryBoundaries(t *testing.T) {
	pool := testsupport.NewDB(t)
	store := adminauth.NewPostgresSessionStore(pool)
	ctx := context.Background()
	now := time.Date(2026, 7, 11, 12, 0, 0, 0, time.UTC)
	create := func(token string, absolute, idle time.Time) {
		t.Helper()
		_, err := store.CreateAdminSession(ctx, adminauth.CreateSessionParams{
			TokenHash: token, CSRFTokenHash: "csrf-" + token,
			CSRFTokenCipher: []byte{1}, CSRFNonce: []byte{2}, CSRFKeyVersion: 1,
			ActorIssuer: "https://id.example.test", ActorSubject: "user-1",
			Roles: []string{"viewer"}, Permissions: []string{"admin.read"},
			CreatedAt: now.Add(-time.Hour), ExpiresAt: absolute, IdleExpiresAt: idle,
		})
		if err != nil {
			t.Fatalf("create session: %v", err)
		}
	}
	create("absolute-boundary", now, now)
	create("idle-boundary", now.Add(time.Hour), now)
	for _, token := range []string{"absolute-boundary", "idle-boundary"} {
		if _, err := store.AuthenticateAdminSession(ctx, adminauth.AuthenticateSessionParams{
			TokenHash: token, UsedAt: now, IdleSeconds: 1800,
		}); err != pgx.ErrNoRows {
			t.Fatalf("token=%s err=%v want pgx.ErrNoRows", token, err)
		}
	}
}

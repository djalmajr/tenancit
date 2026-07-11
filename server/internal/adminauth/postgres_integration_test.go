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

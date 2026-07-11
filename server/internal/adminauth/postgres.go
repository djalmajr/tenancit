package adminauth

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresSessionStore struct {
	pool *pgxpool.Pool
}

func NewPostgresSessionStore(pool *pgxpool.Pool) *PostgresSessionStore {
	return &PostgresSessionStore{pool: pool}
}

func (s *PostgresSessionStore) RevokeAdminSession(ctx context.Context, id uuid.UUID, revokedAt time.Time) error {
	command, err := s.pool.Exec(ctx, `
		UPDATE admin_sessions SET revoked_at = $2 WHERE id = $1 AND revoked_at IS NULL
	`, id, revokedAt)
	if err != nil {
		return err
	}
	if command.RowsAffected() != 1 {
		return errors.New("admin session not found or already revoked")
	}
	return nil
}

func (s *PostgresSessionStore) CreateAdminSession(ctx context.Context, params CreateSessionParams) (StoredSession, error) {
	row := s.pool.QueryRow(ctx, `
		INSERT INTO admin_sessions (
			token_hash, csrf_token_hash, csrf_token_cipher, csrf_nonce, csrf_key_version,
			actor_issuer, actor_subject, actor_label,
			roles, permissions, created_at, last_used_at, expires_at, idle_expires_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $11, $12, $13)
		RETURNING id, token_hash, csrf_token_hash, csrf_token_cipher, csrf_nonce, csrf_key_version,
			actor_issuer, actor_subject,
			actor_label, roles, permissions, created_at, last_used_at,
			expires_at, idle_expires_at, revoked_at
	`, params.TokenHash, params.CSRFTokenHash, params.CSRFTokenCipher, params.CSRFNonce,
		params.CSRFKeyVersion, params.ActorIssuer, params.ActorSubject,
		params.ActorLabel, params.Roles, params.Permissions, params.CreatedAt,
		params.ExpiresAt, params.IdleExpiresAt)
	return scanStoredSession(row)
}

func (s *PostgresSessionStore) AuthenticateAdminSession(ctx context.Context, params AuthenticateSessionParams) (StoredSession, error) {
	row := s.pool.QueryRow(ctx, `
		UPDATE admin_sessions
		SET last_used_at = GREATEST(last_used_at, $1),
			idle_expires_at = LEAST(expires_at, $1 + make_interval(secs => $2::integer))
		WHERE token_hash = $3 AND revoked_at IS NULL
		  AND expires_at > $1 AND idle_expires_at > $1
		RETURNING id, token_hash, csrf_token_hash, csrf_token_cipher, csrf_nonce, csrf_key_version,
			actor_issuer, actor_subject,
			actor_label, roles, permissions, created_at, last_used_at,
			expires_at, idle_expires_at, revoked_at
	`, params.UsedAt, params.IdleSeconds, params.TokenHash)
	return scanStoredSession(row)
}

type rowScanner interface {
	Scan(...any) error
}

func scanStoredSession(row rowScanner) (StoredSession, error) {
	var session StoredSession
	err := row.Scan(
		&session.ID, &session.TokenHash, &session.CSRFTokenHash,
		&session.CSRFTokenCipher, &session.CSRFNonce, &session.CSRFKeyVersion,
		&session.ActorIssuer, &session.ActorSubject, &session.ActorLabel,
		&session.Roles, &session.Permissions, &session.CreatedAt,
		&session.LastUsedAt, &session.ExpiresAt, &session.IdleExpiresAt,
		&session.RevokedAt,
	)
	return session, err
}

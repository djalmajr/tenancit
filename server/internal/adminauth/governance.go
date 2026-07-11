package adminauth

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

var (
	ErrCurrentSessionRequiresLogout = errors.New("current session requires logout")
	ErrSessionNotActive             = errors.New("admin session not found or inactive")
)

type GovernedSession struct {
	ID, Issuer, Subject, Label string
	Roles                      []string
	CreatedAt, LastUsedAt      time.Time
	ExpiresAt, IdleExpiresAt   time.Time
	RevokedAt                  pgtype.Timestamptz
}

func (s *PostgresSessionStore) ListAdminSessions(ctx context.Context) ([]GovernedSession, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, actor_issuer, actor_subject, COALESCE(actor_label, ''), roles,
			created_at, last_used_at, expires_at, idle_expires_at, revoked_at
		FROM admin_sessions
		ORDER BY last_used_at DESC, id DESC
		LIMIT 200
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	sessions := []GovernedSession{}
	for rows.Next() {
		var session GovernedSession
		if err := rows.Scan(
			&session.ID, &session.Issuer, &session.Subject, &session.Label, &session.Roles,
			&session.CreatedAt, &session.LastUsedAt, &session.ExpiresAt, &session.IdleExpiresAt, &session.RevokedAt,
		); err != nil {
			return nil, err
		}
		sessions = append(sessions, session)
	}
	return sessions, rows.Err()
}

func (s *PostgresSessionStore) RevokeOtherAdminSession(ctx context.Context, tx pgx.Tx, id, currentID uuid.UUID, revokedAt time.Time) error {
	if id == currentID {
		return ErrCurrentSessionRequiresLogout
	}
	command, err := tx.Exec(ctx, `
		UPDATE admin_sessions SET revoked_at = $2
		WHERE id = $1 AND revoked_at IS NULL AND expires_at > $2 AND idle_expires_at > $2
	`, id, revokedAt.UTC())
	if err != nil {
		return err
	}
	if command.RowsAffected() != 1 {
		return ErrSessionNotActive
	}
	return nil
}

func (s *PostgresSessionStore) RevokePrincipalSessions(ctx context.Context, tx pgx.Tx, issuer, subject string, currentID uuid.UUID, revokedAt time.Time) (int64, error) {
	command, err := tx.Exec(ctx, `
		UPDATE admin_sessions SET revoked_at = $4
		WHERE actor_issuer = $1 AND actor_subject = $2 AND id <> $3
		  AND revoked_at IS NULL AND expires_at > $4 AND idle_expires_at > $4
	`, issuer, subject, currentID, revokedAt.UTC())
	if err != nil {
		return 0, err
	}
	return command.RowsAffected(), nil
}

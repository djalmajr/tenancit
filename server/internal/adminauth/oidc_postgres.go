package adminauth

import (
	"context"
	"time"
)

func (s *PostgresSessionStore) CreateLoginAttempt(ctx context.Context, params CreateLoginAttemptParams) (StoredLoginAttempt, error) {
	row := s.pool.QueryRow(ctx, `
		INSERT INTO oidc_login_attempts (
			state_hash, nonce_hash, pkce_verifier_cipher, cipher_nonce,
			key_version, redirect_after, created_at, expires_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING state_hash, nonce_hash, pkce_verifier_cipher, cipher_nonce,
			key_version, redirect_after, created_at, expires_at
	`, params.StateHash, params.NonceHash, params.PKCEVerifierCipher, params.CipherNonce,
		params.KeyVersion, params.RedirectAfter, params.CreatedAt, params.ExpiresAt)
	return scanLoginAttempt(row)
}

func (s *PostgresSessionStore) ConsumeLoginAttempt(ctx context.Context, stateHash string, now time.Time) (StoredLoginAttempt, error) {
	row := s.pool.QueryRow(ctx, `
		DELETE FROM oidc_login_attempts
		WHERE state_hash = $1 AND expires_at > $2 AND consumed_at IS NULL
		RETURNING state_hash, nonce_hash, pkce_verifier_cipher, cipher_nonce,
			key_version, redirect_after, created_at, expires_at
	`, stateHash, now)
	return scanLoginAttempt(row)
}

func scanLoginAttempt(row rowScanner) (StoredLoginAttempt, error) {
	var attempt StoredLoginAttempt
	err := row.Scan(
		&attempt.StateHash, &attempt.NonceHash, &attempt.PKCEVerifierCipher,
		&attempt.CipherNonce, &attempt.KeyVersion, &attempt.RedirectAfter,
		&attempt.CreatedAt, &attempt.ExpiresAt,
	)
	return attempt, err
}

// Package idempotency implements transactional replay for selected admin mutations.
package idempotency

import (
	"context"
	"crypto/sha256"
	"errors"
	"time"

	appcrypto "github.com/djalmajr/tenancit/server/internal/crypto"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var (
	ErrInvalid             = errors.New("invalid idempotency request")
	ErrFingerprintMismatch = errors.New("idempotency fingerprint mismatch")
	ErrInProgress          = errors.New("idempotency request is in progress")
	ErrExpired             = errors.New("idempotency request expired")
)

type Actor struct {
	Kind, Issuer, Subject string
}

type Request struct {
	Actor       Actor
	Operation   string
	Key         uuid.UUID
	Fingerprint [32]byte
	TTL         time.Duration
}

type Replay struct {
	Found       bool
	Status      int
	ContentType string
	Body        []byte
}

type Store struct {
	cryptor *appcrypto.Cryptor
	now     func() time.Time
}

func NewStore(cryptor *appcrypto.Cryptor, now func() time.Time) *Store {
	if now == nil {
		now = time.Now
	}
	return &Store{cryptor: cryptor, now: now}
}

func Fingerprint(operation string, canonicalPayload []byte) [32]byte {
	digest := sha256.New()
	_, _ = digest.Write([]byte(operation))
	_, _ = digest.Write([]byte{0})
	_, _ = digest.Write(canonicalPayload)
	var result [sha256.Size]byte
	copy(result[:], digest.Sum(nil))
	return result
}

func (s *Store) Begin(ctx context.Context, tx pgx.Tx, request Request) (Replay, error) {
	if s == nil || s.cryptor == nil || tx == nil || request.Key == uuid.Nil || request.Actor.Subject == "" || request.Operation == "" || request.TTL <= 0 ||
		(request.Actor.Kind != "shared_admin_token" && request.Actor.Kind != "oidc_user") || (request.Actor.Kind == "oidc_user") != (request.Actor.Issuer != "") {
		return Replay{}, ErrInvalid
	}
	now := s.now().UTC()
	command, err := tx.Exec(ctx, `INSERT INTO admin_idempotency_records
		(actor_kind,actor_issuer,actor_subject,operation,idempotency_key,request_fingerprint,created_at,expires_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT DO NOTHING`,
		request.Actor.Kind, request.Actor.Issuer, request.Actor.Subject, request.Operation,
		request.Key, request.Fingerprint[:], now, now.Add(request.TTL))
	if err != nil {
		return Replay{}, err
	}
	if command.RowsAffected() == 1 {
		return Replay{}, nil
	}
	var fingerprint, cipher, nonce []byte
	var status, contentType string
	var httpStatus *int16
	var keyVersion *int16
	var expiresAt time.Time
	err = tx.QueryRow(ctx, `SELECT request_fingerprint,status,http_status,content_type,response_cipher,nonce,key_version,expires_at
		FROM admin_idempotency_records
		WHERE actor_kind=$1 AND actor_issuer=$2 AND actor_subject=$3 AND operation=$4 AND idempotency_key=$5 FOR UPDATE`,
		request.Actor.Kind, request.Actor.Issuer, request.Actor.Subject, request.Operation, request.Key).
		Scan(&fingerprint, &status, &httpStatus, &contentType, &cipher, &nonce, &keyVersion, &expiresAt)
	if err != nil {
		return Replay{}, err
	}
	if !equalFingerprint(fingerprint, request.Fingerprint) {
		return Replay{}, ErrFingerprintMismatch
	}
	if !expiresAt.After(now) {
		return Replay{}, ErrExpired
	}
	if status != "completed" || httpStatus == nil || keyVersion == nil {
		return Replay{}, ErrInProgress
	}
	body, err := s.cryptor.DecryptBytes(appcrypto.Encrypted{Cipher: cipher, Nonce: nonce, KeyVersion: int(*keyVersion)})
	if err != nil {
		return Replay{}, err
	}
	return Replay{Found: true, Status: int(*httpStatus), ContentType: contentType, Body: body}, nil
}

func (s *Store) Complete(ctx context.Context, tx pgx.Tx, request Request, status int, contentType string, body []byte) error {
	if status < 200 || status > 299 || len(body) == 0 || len(contentType) > 128 {
		return ErrInvalid
	}
	encrypted, err := s.cryptor.EncryptBytes(body)
	if err != nil {
		return err
	}
	command, err := tx.Exec(ctx, `UPDATE admin_idempotency_records SET
		status='completed',http_status=$6,content_type=$7,response_cipher=$8,nonce=$9,key_version=$10,completed_at=clock_timestamp()
		WHERE actor_kind=$1 AND actor_issuer=$2 AND actor_subject=$3 AND operation=$4 AND idempotency_key=$5 AND status='processing' AND request_fingerprint=$11`,
		request.Actor.Kind, request.Actor.Issuer, request.Actor.Subject, request.Operation, request.Key,
		int16(status), contentType, encrypted.Cipher, encrypted.Nonce, int16(encrypted.KeyVersion), request.Fingerprint[:])
	if err != nil {
		return err
	}
	if command.RowsAffected() != 1 {
		return ErrInvalid
	}
	return nil
}

func CleanupExpired(ctx context.Context, tx pgx.Tx, now time.Time, limit int) (int64, error) {
	if tx == nil || limit < 1 || limit > 10000 {
		return 0, ErrInvalid
	}
	command, err := tx.Exec(ctx, `DELETE FROM admin_idempotency_records WHERE id IN (
		SELECT id FROM admin_idempotency_records WHERE expires_at<=$1 ORDER BY expires_at FOR UPDATE SKIP LOCKED LIMIT $2
	)`, now.UTC(), limit)
	if err != nil {
		return 0, err
	}
	return command.RowsAffected(), nil
}

func equalFingerprint(stored []byte, expected [32]byte) bool {
	if len(stored) != len(expected) {
		return false
	}
	var difference byte
	for index := range expected {
		difference |= stored[index] ^ expected[index]
	}
	return difference == 0
}

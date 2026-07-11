package rewrap

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	appcrypto "github.com/djalmajr/tenancit/server/internal/crypto"
	"github.com/djalmajr/tenancit/server/internal/telemetry"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func (r *Runner) processBatch(ctx context.Context, targetVersion, batchSize int) (int, error) {
	tx, err := r.DB.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return 0, fmt.Errorf("begin rewrap batch: %w", err)
	}
	defer tx.Rollback(ctx)
	rows, err := tx.Query(ctx, `SELECT id,value_plain,value_cipher,nonce,key_version FROM tenant_resource_values
		WHERE value_cipher IS NOT NULL AND key_version IS DISTINCT FROM $1
		ORDER BY id LIMIT $2 FOR UPDATE SKIP LOCKED`, targetVersion, batchSize)
	if err != nil {
		return 0, fmt.Errorf("select rewrap batch: %w", err)
	}
	batch := make([]encryptedRow, 0, batchSize)
	versionCounts := make(map[int]int)
	for rows.Next() {
		var row encryptedRow
		var version pgtype.Int4
		if err := rows.Scan(&row.ID, &row.Plain, &row.Cipher, &row.Nonce, &version); err != nil {
			rows.Close()
			return 0, fmt.Errorf("scan rewrap candidate: %w", err)
		}
		if version.Valid {
			value := version.Int32
			row.KeyVersion = &value
		}
		batch = append(batch, row)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, fmt.Errorf("iterate rewrap candidates: %w", err)
	}
	rows.Close()
	for _, row := range batch {
		if err := r.authenticateRow(row); err != nil {
			return 0, err
		}
		version := int(*row.KeyVersion)
		versionCounts[version]++
		plaintext, err := r.Cryptor.DecryptBytes(appcrypto.Encrypted{Cipher: row.Cipher, Nonce: row.Nonce, KeyVersion: version})
		if err != nil {
			return 0, ErrAuthentication
		}
		rewrapped, err := r.Cryptor.EncryptBytes(plaintext)
		if err != nil {
			clear(plaintext)
			return 0, fmt.Errorf("encrypt rewrapped value: %w", err)
		}
		verified, err := r.Cryptor.DecryptBytes(rewrapped)
		if err != nil || !bytes.Equal(plaintext, verified) {
			clear(plaintext)
			clear(verified)
			return 0, ErrVerification
		}
		clear(plaintext)
		clear(verified)
		command, err := tx.Exec(ctx, `UPDATE tenant_resource_values SET value_cipher=$1,nonce=$2,key_version=$3
			WHERE id=$4 AND key_version=$5 AND value_cipher=$6 AND nonce=$7`,
			rewrapped.Cipher, rewrapped.Nonce, targetVersion, row.ID, version, row.Cipher, row.Nonce)
		if err != nil {
			return 0, fmt.Errorf("update rewrapped value: %w", err)
		}
		if command.RowsAffected() != 1 {
			return 0, ErrCASConflict
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit rewrap batch: %w", err)
	}
	for version, count := range versionCounts {
		telemetry.RecordRewrapRows(ctx, version, targetVersion, "success", count)
	}
	return len(batch), nil
}

func (r *Runner) remaining(ctx context.Context, targetVersion int) (int64, error) {
	var count int64
	if err := r.DB.QueryRow(ctx, `SELECT count(*) FROM tenant_resource_values WHERE value_cipher IS NOT NULL AND key_version IS DISTINCT FROM $1`, targetVersion).Scan(&count); err != nil {
		return 0, fmt.Errorf("count remaining rewrap rows: %w", err)
	}
	return count, nil
}

func (r *Runner) hasSafetyEvidence(ctx context.Context) (bool, error) {
	var backup, restore bool
	err := r.DB.QueryRow(ctx, `SELECT
		EXISTS(SELECT 1 FROM operational_reports WHERE kind='backup' AND status='healthy' AND fresh_until>clock_timestamp()),
		EXISTS(SELECT 1 FROM operational_reports WHERE kind='restore' AND status='healthy' AND fresh_until>clock_timestamp())`).Scan(&backup, &restore)
	if err != nil {
		return false, fmt.Errorf("check backup and restore evidence: %w", err)
	}
	return backup && restore, nil
}

func metricFailureReason(err error) string {
	switch {
	case errors.Is(err, ErrCampaignLocked):
		return "campaign_locked"
	case errors.Is(err, ErrInvalidConfig):
		return "invalid_config"
	case errors.Is(err, ErrSafetyEvidence):
		return "safety_evidence"
	case errors.Is(err, ErrMalformedCiphertext):
		return "malformed"
	case errors.Is(err, ErrMissingKeyVersion):
		return "missing_key"
	case errors.Is(err, ErrAuthentication):
		return "authentication"
	case errors.Is(err, ErrVerification):
		return "verification"
	case errors.Is(err, ErrCASConflict):
		return "cas_conflict"
	case errors.Is(err, ErrNoProgress):
		return "no_progress"
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return "canceled"
	default:
		return "internal"
	}
}

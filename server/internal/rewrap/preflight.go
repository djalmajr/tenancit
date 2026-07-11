package rewrap

import (
	"context"
	"fmt"

	appcrypto "github.com/djalmajr/tenancit/server/internal/crypto"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func (r *Runner) inventory(ctx context.Context, database queryer) ([]InventoryItem, error) {
	rows, err := database.Query(ctx, `SELECT key_version, count(*) FROM tenant_resource_values WHERE value_cipher IS NOT NULL GROUP BY key_version ORDER BY key_version NULLS FIRST`)
	if err != nil {
		return nil, fmt.Errorf("inventory encrypted rows: %w", err)
	}
	defer rows.Close()
	items := []InventoryItem{}
	for rows.Next() {
		var version pgtype.Int4
		var count int64
		if err := rows.Scan(&version, &count); err != nil {
			return nil, fmt.Errorf("scan encrypted inventory: %w", err)
		}
		var pointer *int32
		if version.Valid {
			value := version.Int32
			pointer = &value
		}
		items = append(items, InventoryItem{KeyVersion: pointer, Rows: count})
	}
	return items, rows.Err()
}

func (r *Runner) authenticateAll(ctx context.Context, database queryer, pageSize int, summary *Summary) error {
	var cursor uuid.UUID
	hasCursor := false
	for {
		rows, err := database.Query(ctx, `SELECT id,value_plain,value_cipher,nonce,key_version FROM tenant_resource_values
			WHERE value_cipher IS NOT NULL AND (NOT $1::boolean OR id > $2::uuid)
			ORDER BY id LIMIT $3`, hasCursor, cursor, pageSize)
		if err != nil {
			return fmt.Errorf("page encrypted rows: %w", err)
		}
		page := make([]encryptedRow, 0, pageSize)
		for rows.Next() {
			var row encryptedRow
			var version pgtype.Int4
			if err := rows.Scan(&row.ID, &row.Plain, &row.Cipher, &row.Nonce, &version); err != nil {
				rows.Close()
				return fmt.Errorf("scan encrypted row: %w", err)
			}
			if version.Valid {
				value := version.Int32
				row.KeyVersion = &value
			}
			page = append(page, row)
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return fmt.Errorf("iterate encrypted rows: %w", err)
		}
		rows.Close()
		for _, row := range page {
			if err := r.authenticateRow(row); err != nil {
				return err
			}
			summary.Scanned++
			cursor = row.ID
			hasCursor = true
		}
		if len(page) < pageSize {
			return nil
		}
	}
}

func (r *Runner) authenticateRow(row encryptedRow) error {
	if row.Plain != nil || row.KeyVersion == nil || *row.KeyVersion <= 0 {
		return ErrMalformedCiphertext
	}
	version := int(*row.KeyVersion)
	nonceSize, ok := r.Cryptor.NonceSize(version)
	if !ok {
		return ErrMissingKeyVersion
	}
	overhead, _ := r.Cryptor.Overhead(version)
	if len(row.Nonce) != nonceSize || len(row.Cipher) < overhead {
		return ErrMalformedCiphertext
	}
	plaintext, err := r.Cryptor.DecryptBytes(appcrypto.Encrypted{Cipher: row.Cipher, Nonce: row.Nonce, KeyVersion: version})
	if err != nil {
		return ErrAuthentication
	}
	clear(plaintext)
	return nil
}

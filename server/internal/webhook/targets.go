package webhook

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"strings"
	"time"

	appcrypto "github.com/djalmajr/tenancit/server/internal/crypto"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrTargetNotFound = errors.New("webhook target not found")

type Target struct {
	ID                  uuid.UUID  `json:"id"`
	Name                string     `json:"name"`
	Format              string     `json:"format"`
	Status              string     `json:"status"`
	Endpoint            string     `json:"endpoint"`
	ConsecutiveFailures int32      `json:"consecutive_failures"`
	CircuitOpenUntil    *time.Time `json:"circuit_open_until,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

type CreatedTarget struct {
	Target
	SigningSecret string `json:"signing_secret"`
}

type TargetRepository struct {
	pool              *pgxpool.Pool
	cryptor           *appcrypto.Cryptor
	random            io.Reader
	resolver          Resolver
	allowLoopbackHTTP bool
}

func NewTargetRepository(pool *pgxpool.Pool, cryptor *appcrypto.Cryptor, random io.Reader, resolver Resolver, allowLoopbackHTTP bool) *TargetRepository {
	if random == nil {
		random = rand.Reader
	}
	if resolver == nil {
		resolver = net.DefaultResolver
	}
	return &TargetRepository{pool: pool, cryptor: cryptor, random: random, resolver: resolver, allowLoopbackHTTP: allowLoopbackHTTP}
}

func (r *TargetRepository) Create(ctx context.Context, name, rawURL, format string) (CreatedTarget, error) {
	return r.create(ctx, r.pool, name, rawURL, format)
}

func (r *TargetRepository) CreateTx(ctx context.Context, tx pgx.Tx, name, rawURL, format string) (CreatedTarget, error) {
	return r.create(ctx, tx, name, rawURL, format)
}

type targetQueryer interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func (r *TargetRepository) create(ctx context.Context, queryer targetQueryer, name, rawURL, format string) (CreatedTarget, error) {
	name, format = strings.TrimSpace(name), strings.TrimSpace(format)
	if r == nil || r.pool == nil || r.cryptor == nil || name == "" || !validFormat(format) {
		return CreatedTarget{}, errors.New("valid webhook target is required")
	}
	endpoint, err := ValidateEndpoint(ctx, rawURL, r.allowLoopbackHTTP, r.resolver)
	if err != nil {
		return CreatedTarget{}, err
	}
	secretBytes := make([]byte, 32)
	if _, err := io.ReadFull(r.random, secretBytes); err != nil {
		return CreatedTarget{}, fmt.Errorf("generate signing secret: %w", err)
	}
	secret := base64.RawURLEncoding.EncodeToString(secretBytes)
	urlEncrypted, err := r.cryptor.Encrypt(endpoint.String())
	if err != nil {
		return CreatedTarget{}, err
	}
	secretEncrypted, err := r.cryptor.Encrypt(secret)
	if err != nil {
		return CreatedTarget{}, err
	}
	var target Target
	err = queryer.QueryRow(ctx, `
		INSERT INTO webhook_targets (
			name, format, url_cipher, url_nonce, url_key_version,
			signing_secret_cipher, signing_secret_nonce, signing_secret_key_version,
			allow_loopback_http
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, name, format, status, consecutive_failures, circuit_open_until, created_at, updated_at
	`, name, format, urlEncrypted.Cipher, urlEncrypted.Nonce, int16(urlEncrypted.KeyVersion),
		secretEncrypted.Cipher, secretEncrypted.Nonce, int16(secretEncrypted.KeyVersion), r.allowLoopbackHTTP,
	).Scan(&target.ID, &target.Name, &target.Format, &target.Status, &target.ConsecutiveFailures, &target.CircuitOpenUntil, &target.CreatedAt, &target.UpdatedAt)
	if err != nil {
		return CreatedTarget{}, err
	}
	target.Endpoint = endpoint.Scheme + "://" + endpoint.Host
	return CreatedTarget{Target: target, SigningSecret: secret}, nil
}

func (r *TargetRepository) List(ctx context.Context) ([]Target, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, name, format, status, url_cipher, url_nonce, url_key_version,
			consecutive_failures, circuit_open_until, created_at, updated_at
		FROM webhook_targets ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	targets := []Target{}
	for rows.Next() {
		var target Target
		var cipher, nonce []byte
		var version int16
		if err := rows.Scan(&target.ID, &target.Name, &target.Format, &target.Status, &cipher, &nonce, &version, &target.ConsecutiveFailures, &target.CircuitOpenUntil, &target.CreatedAt, &target.UpdatedAt); err != nil {
			return nil, err
		}
		rawURL, err := r.cryptor.Decrypt(appcrypto.Encrypted{Cipher: cipher, Nonce: nonce, KeyVersion: int(version)})
		if err != nil {
			return nil, err
		}
		endpoint, err := neturl(rawURL)
		if err != nil {
			return nil, err
		}
		target.Endpoint = endpoint
		targets = append(targets, target)
	}
	return targets, rows.Err()
}

func (r *TargetRepository) SetStatus(ctx context.Context, id uuid.UUID, status string) error {
	return r.setStatus(ctx, r.pool, id, status)
}

func (r *TargetRepository) SetStatusTx(ctx context.Context, tx pgx.Tx, id uuid.UUID, status string) error {
	return r.setStatus(ctx, tx, id, status)
}

type targetExecer interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

func (r *TargetRepository) setStatus(ctx context.Context, execer targetExecer, id uuid.UUID, status string) error {
	if status != "active" && status != "disabled" {
		return errors.New("invalid webhook target status")
	}
	command, err := execer.Exec(ctx, `UPDATE webhook_targets SET status = $2, updated_at = clock_timestamp() WHERE id = $1`, id, status)
	if err == nil && command.RowsAffected() == 0 {
		return ErrTargetNotFound
	}
	return err
}

func validFormat(value string) bool {
	return value == "generic" || value == "slack" || value == "discord" || value == "teams"
}

func neturl(rawURL string) (string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Host == "" {
		return "", errors.New("stored webhook URL is invalid")
	}
	return parsed.Scheme + "://" + parsed.Host, nil
}

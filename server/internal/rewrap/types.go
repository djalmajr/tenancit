package rewrap

import (
	"context"
	"errors"
	"log/slog"
	"time"

	appcrypto "github.com/djalmajr/tenancit/server/internal/crypto"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const advisoryLockID int64 = 6072342442203466068

var (
	ErrCampaignLocked      = errors.New("rewrap campaign already running")
	ErrInvalidConfig       = errors.New("invalid rewrap configuration")
	ErrSafetyEvidence      = errors.New("fresh backup and restore evidence required")
	ErrMalformedCiphertext = errors.New("malformed encrypted row")
	ErrMissingKeyVersion   = errors.New("encrypted row uses an unloaded key version")
	ErrAuthentication      = errors.New("ciphertext authentication failed")
	ErrVerification        = errors.New("rewrap verification failed")
	ErrCASConflict         = errors.New("rewrap compare-and-swap conflict")
	ErrNoProgress          = errors.New("rewrap made no progress")
)

type Config struct {
	TargetVersion     int
	BatchSize         int
	DryRun            bool
	ConfirmedWrite    bool
	MaxDuration       time.Duration
	NoProgressTimeout time.Duration
	PollInterval      time.Duration
	JobID             string
}

type InventoryItem struct {
	KeyVersion *int32 `json:"key_version"`
	Rows       int64  `json:"rows"`
}

type Summary struct {
	JobID         string
	TargetVersion int
	DryRun        bool
	Scanned       int64
	Rewrapped     int64
	Batches       int64
	LockedRetries int64
	Remaining     int64
	Inventory     []InventoryItem
	StartedAt     time.Time
	CompletedAt   time.Time
}

type Reporter interface {
	Report(context.Context, Summary, string) error
}

type Runner struct {
	DB       *pgxpool.Pool
	Cryptor  *appcrypto.Cryptor
	Reporter Reporter
	Now      func() time.Time
	Logger   *slog.Logger
}

type encryptedRow struct {
	ID         uuid.UUID
	Plain      *string
	Cipher     []byte
	Nonce      []byte
	KeyVersion *int32
}

type queryer interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

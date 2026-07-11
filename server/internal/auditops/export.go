package auditops

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	appcrypto "github.com/djalmajr/tenancit/server/internal/crypto"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	MaxExportWindow = 31 * 24 * time.Hour
	SyncExportLimit = 1000
	MaxExportRows   = 100000
)

var (
	ErrInvalidExport       = errors.New("invalid audit export")
	ErrExportNotReady      = errors.New("audit export is not ready")
	ErrExportConsumed      = errors.New("audit export was already downloaded")
	ErrExportExpired       = errors.New("audit export expired")
	ErrIdempotencyMismatch = errors.New("audit export idempotency mismatch")
)

type ExportFilter struct {
	From         time.Time `json:"from"`
	To           time.Time `json:"to"`
	ActorKind    string    `json:"actor_kind,omitempty"`
	ActorSubject string    `json:"actor_subject,omitempty"`
	Action       string    `json:"action,omitempty"`
	TargetType   string    `json:"target_type,omitempty"`
	TargetID     string    `json:"target_id,omitempty"`
	RequestID    string    `json:"request_id,omitempty"`
	Result       string    `json:"result,omitempty"`
}

type ExportActor struct{ Kind, Issuer, Subject, Label string }

type DownloadAudit struct {
	RequestID, Method, RouteTemplate string
}

type ExportJob struct {
	ID             uuid.UUID    `json:"id"`
	IdempotencyKey uuid.UUID    `json:"idempotency_key"`
	Filters        ExportFilter `json:"filters"`
	Format         string       `json:"format"`
	Status         string       `json:"status"`
	RowCount       *int64       `json:"row_count,omitempty"`
	CreatedAt      time.Time    `json:"created_at"`
	CompletedAt    *time.Time   `json:"completed_at,omitempty"`
	ExpiresAt      time.Time    `json:"expires_at"`
	DownloadedAt   *time.Time   `json:"downloaded_at,omitempty"`
}

type ExportRepository struct {
	pool    *pgxpool.Pool
	cryptor *appcrypto.Cryptor
	now     func() time.Time
	sink    ArchiveSink
}

type ArchiveMetadata struct {
	JobID       uuid.UUID
	Format      string
	Rows        int64
	SHA256      [32]byte
	GeneratedAt time.Time
}
type ArchiveSink interface {
	Archive(context.Context, ArchiveMetadata, []byte) error
}

func NewExportRepository(pool *pgxpool.Pool, cryptor *appcrypto.Cryptor, now func() time.Time) *ExportRepository {
	if now == nil {
		now = time.Now
	}
	return &ExportRepository{pool: pool, cryptor: cryptor, now: now}
}

func (r *ExportRepository) SetArchiveSink(sink ArchiveSink) { r.sink = sink }

func ValidateExport(filter ExportFilter, format string) error {
	if filter.From.IsZero() || filter.To.IsZero() || !filter.To.After(filter.From) || filter.To.Sub(filter.From) > MaxExportWindow || (format != "csv" && format != "jsonl") {
		return ErrInvalidExport
	}
	if filter.Result != "" && filter.Result != "success" && filter.Result != "denied" && filter.Result != "error" {
		return ErrInvalidExport
	}
	if filter.ActorKind != "" && filter.ActorKind != "shared_admin_token" && filter.ActorKind != "break_glass" && filter.ActorKind != "oidc_user" && filter.ActorKind != "unauthenticated" {
		return ErrInvalidExport
	}
	for _, value := range []string{filter.ActorSubject, filter.Action, filter.TargetType, filter.TargetID, filter.RequestID} {
		if len(value) > 256 || strings.ContainsAny(value, "\r\n\x00") {
			return ErrInvalidExport
		}
	}
	return nil
}

func (r *ExportRepository) Count(ctx context.Context, filter ExportFilter) (int64, error) {
	var count int64
	err := r.pool.QueryRow(ctx, exportSelect(true), exportArgs(filter)...).Scan(&count)
	return count, err
}

func (r *ExportRepository) Create(ctx context.Context, actor ExportActor, key uuid.UUID, filter ExportFilter, format string, synchronous bool, audit DownloadAudit) (ExportJob, error) {
	var job ExportJob
	if r == nil || r.pool == nil || r.cryptor == nil || key == uuid.Nil || actor.Subject == "" || audit.RequestID == "" || audit.Method == "" || audit.RouteTemplate == "" || ValidateExport(filter, format) != nil {
		return job, ErrInvalidExport
	}
	encoded, err := json.Marshal(filter)
	if err != nil {
		return job, err
	}
	fingerprint := sha256.Sum256(append(append([]byte(nil), encoded...), []byte("\x00"+format)...))
	status := "pending"
	if synchronous {
		status = "processing"
	}
	expires := r.now().UTC().Add(24 * time.Hour)
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return job, err
	}
	defer tx.Rollback(ctx)
	var inserted bool
	err = tx.QueryRow(ctx, `INSERT INTO audit_export_jobs
		(idempotency_key,request_fingerprint,requested_by_kind,requested_by_issuer,requested_by_subject,filters,format,status,expires_at)
		VALUES ($1,$2,$3,NULLIF($4,''),$5,$6,$7,$8,$9)
		ON CONFLICT (requested_by_kind,requested_by_subject,idempotency_key) DO UPDATE SET idempotency_key=EXCLUDED.idempotency_key
		WHERE audit_export_jobs.request_fingerprint=EXCLUDED.request_fingerprint
		RETURNING id,idempotency_key,filters,format,status,row_count,created_at,completed_at,expires_at,downloaded_at,(xmax=0)`,
		key, fingerprint[:], actor.Kind, actor.Issuer, actor.Subject, encoded, format, status, expires).Scan(&job.ID, &job.IdempotencyKey, &encoded, &job.Format, &job.Status, &job.RowCount, &job.CreatedAt, &job.CompletedAt, &job.ExpiresAt, &job.DownloadedAt, &inserted)
	if errors.Is(err, pgx.ErrNoRows) {
		return job, ErrIdempotencyMismatch
	}
	if err != nil {
		return job, err
	}
	if inserted {
		metadata, _ := json.Marshal(struct {
			Format      string    `json:"format"`
			Synchronous bool      `json:"synchronous"`
			From        time.Time `json:"from"`
			To          time.Time `json:"to"`
		}{format, synchronous, filter.From, filter.To})
		_, err = tx.Exec(ctx, `INSERT INTO admin_audit_events(request_id,actor_kind,actor_issuer,actor_subject,actor_label,action,target_type,target_id,result,http_method,route_template,http_status,metadata) VALUES ($1,$2,NULLIF($3,''),$4,NULLIF($5,''),'audit.export_requested','audit_export',$6,'success',$7,$8,202,$9)`, audit.RequestID, actor.Kind, actor.Issuer, actor.Subject, actor.Label, job.ID.String(), audit.Method, audit.RouteTemplate, metadata)
		if err != nil {
			return job, err
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return job, err
	}
	if err := json.Unmarshal(encoded, &job.Filters); err != nil {
		return job, err
	}
	if synchronous && job.Status == "processing" {
		if err := r.process(ctx, job.ID, filter, format); err != nil {
			return job, err
		}
		return r.Get(ctx, actor, job.ID)
	}
	return job, nil
}

func (r *ExportRepository) Get(ctx context.Context, actor ExportActor, id uuid.UUID) (ExportJob, error) {
	var job ExportJob
	var raw []byte
	err := r.pool.QueryRow(ctx, `SELECT id,idempotency_key,filters,format,status,row_count,created_at,completed_at,expires_at,downloaded_at FROM audit_export_jobs WHERE id=$1 AND requested_by_kind=$2 AND requested_by_subject=$3`, id, actor.Kind, actor.Subject).Scan(&job.ID, &job.IdempotencyKey, &raw, &job.Format, &job.Status, &job.RowCount, &job.CreatedAt, &job.CompletedAt, &job.ExpiresAt, &job.DownloadedAt)
	if err != nil {
		return job, err
	}
	err = json.Unmarshal(raw, &job.Filters)
	return job, err
}

func (r *ExportRepository) ProcessPending(ctx context.Context) (bool, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)
	var id uuid.UUID
	var raw []byte
	var format string
	err = tx.QueryRow(ctx, `UPDATE audit_export_jobs SET status='processing',started_at=clock_timestamp() WHERE id=(SELECT id FROM audit_export_jobs WHERE status='pending' AND expires_at>clock_timestamp() ORDER BY created_at FOR UPDATE SKIP LOCKED LIMIT 1) RETURNING id,filters,format`).Scan(&id, &raw, &format)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if err = tx.Commit(ctx); err != nil {
		return false, err
	}
	var filter ExportFilter
	if err = json.Unmarshal(raw, &filter); err != nil {
		return true, r.fail(ctx, id, "invalid_filters")
	}
	return true, r.process(ctx, id, filter, format)
}

// Consume returns a ready export exactly once. The payload is decrypted only
// while the row is locked; recording the access and destroying the ciphertext
// commit atomically before bytes are returned to the caller.
func (r *ExportRepository) Consume(ctx context.Context, actor ExportActor, id uuid.UUID, audit DownloadAudit) ([]byte, string, error) {
	if actor.Subject == "" || audit.RequestID == "" || audit.Method == "" || audit.RouteTemplate == "" {
		return nil, "", ErrInvalidExport
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, "", err
	}
	defer tx.Rollback(ctx)
	var format, status string
	var cipher, nonce []byte
	var keyVersion *int16
	var expires time.Time
	var downloaded *time.Time
	var rowCount *int64
	err = tx.QueryRow(ctx, `SELECT format,status,payload_cipher,nonce,key_version,expires_at,downloaded_at,row_count
		FROM audit_export_jobs WHERE id=$1 AND requested_by_kind=$2 AND requested_by_subject=$3 FOR UPDATE`, id, actor.Kind, actor.Subject).Scan(&format, &status, &cipher, &nonce, &keyVersion, &expires, &downloaded, &rowCount)
	if err != nil {
		return nil, "", err
	}
	if downloaded != nil {
		return nil, "", ErrExportConsumed
	}
	if !expires.After(r.now().UTC()) {
		return nil, "", ErrExportExpired
	}
	if status != "ready" {
		return nil, "", ErrExportNotReady
	}
	if keyVersion == nil || len(cipher) == 0 || len(nonce) == 0 {
		return nil, "", ErrExportNotReady
	}
	payload, err := r.cryptor.DecryptBytes(appcrypto.Encrypted{Cipher: cipher, Nonce: nonce, KeyVersion: int(*keyVersion)})
	if err != nil {
		return nil, "", err
	}
	command, err := tx.Exec(ctx, `UPDATE audit_export_jobs SET status='expired',downloaded_at=clock_timestamp(),payload_cipher=NULL,nonce=NULL,key_version=NULL WHERE id=$1 AND status='ready' AND downloaded_at IS NULL`, id)
	if err != nil || command.RowsAffected() != 1 {
		clear(payload)
		if err == nil {
			err = ErrExportConsumed
		}
		return nil, "", err
	}
	metadata, _ := json.Marshal(struct {
		Format   string `json:"format"`
		RowCount *int64 `json:"row_count,omitempty"`
	}{Format: format, RowCount: rowCount})
	_, err = tx.Exec(ctx, `INSERT INTO admin_audit_events
		(request_id,actor_kind,actor_issuer,actor_subject,actor_label,action,target_type,target_id,result,http_method,route_template,http_status,metadata)
		VALUES ($1,$2,NULLIF($3,''),$4,NULLIF($5,''),'audit.export_downloaded','audit_export',$6,'success',$7,$8,200,$9)`,
		audit.RequestID, actor.Kind, actor.Issuer, actor.Subject, actor.Label, id.String(), audit.Method, audit.RouteTemplate, metadata)
	if err != nil {
		clear(payload)
		return nil, "", err
	}
	if err := tx.Commit(ctx); err != nil {
		clear(payload)
		return nil, "", err
	}
	return payload, format, nil
}

func (r *ExportRepository) Expire(ctx context.Context) (int64, error) {
	command, err := r.pool.Exec(ctx, `UPDATE audit_export_jobs SET status='expired',payload_cipher=NULL,nonce=NULL,key_version=NULL
		WHERE expires_at<=$1 AND status IN ('pending','processing','ready')`, r.now().UTC())
	if err != nil {
		return 0, err
	}
	return command.RowsAffected(), nil
}

func (r *ExportRepository) process(ctx context.Context, id uuid.UUID, filter ExportFilter, format string) error {
	payload, rows, err := r.generate(ctx, filter, format)
	if err != nil {
		_ = r.fail(ctx, id, "generation_failed")
		return err
	}
	if r.sink != nil {
		metadata := ArchiveMetadata{JobID: id, Format: format, Rows: rows, SHA256: sha256.Sum256(payload), GeneratedAt: r.now().UTC()}
		if err := r.sink.Archive(ctx, metadata, payload); err != nil {
			clear(payload)
			_ = r.fail(ctx, id, "archive_failed")
			return fmt.Errorf("archive audit export: %w", err)
		}
	}
	encrypted, err := r.cryptor.EncryptBytes(payload)
	clear(payload)
	if err != nil {
		_ = r.fail(ctx, id, "encryption_failed")
		return err
	}
	_, err = r.pool.Exec(ctx, `UPDATE audit_export_jobs SET status='ready',row_count=$2,payload_cipher=$3,nonce=$4,key_version=$5,completed_at=clock_timestamp() WHERE id=$1 AND status='processing'`, id, rows, encrypted.Cipher, encrypted.Nonce, encrypted.KeyVersion)
	return err
}

func (r *ExportRepository) fail(ctx context.Context, id uuid.UUID, code string) error {
	_, err := r.pool.Exec(ctx, `UPDATE audit_export_jobs SET status='failed',failure_code=$2,completed_at=clock_timestamp() WHERE id=$1 AND status='processing'`, id, code)
	return err
}

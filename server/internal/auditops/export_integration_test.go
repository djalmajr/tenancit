package auditops

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	appcrypto "github.com/djalmajr/tenancit/server/internal/crypto"
	"github.com/djalmajr/tenancit/server/internal/testsupport"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestSynchronousExportIsEncryptedIdempotentAndConsumedOnce(t *testing.T) {
	pool := testsupport.NewDB(t)
	ctx := context.Background()
	now := time.Date(2026, 7, 11, 12, 0, 0, 0, time.UTC)
	repository := NewExportRepository(pool, exportCryptor(t), func() time.Time { return now })
	seedAuditExportEvent(t, pool, now.Add(-time.Hour), "audit-canary-action")
	actor := ExportActor{Kind: "oidc_user", Issuer: "https://issuer.test", Subject: "auditor-1", Label: "Auditor"}
	filter := ExportFilter{From: now.Add(-24 * time.Hour), To: now, Action: "audit-canary-action"}
	key := uuid.New()

	audit := DownloadAudit{RequestID: "create-sync", Method: "POST", RouteTemplate: "/v1/admin/audit-exports"}
	job, err := repository.Create(ctx, actor, key, filter, "csv", true, audit)
	if err != nil {
		t.Fatalf("create synchronous export: %v", err)
	}
	if job.Status != "ready" || job.RowCount == nil || *job.RowCount != 1 {
		t.Fatalf("job=%+v", job)
	}
	retry, err := repository.Create(ctx, actor, key, filter, "csv", true, audit)
	if err != nil || retry.ID != job.ID {
		t.Fatalf("idempotent retry=%+v err=%v", retry, err)
	}
	mismatch := filter
	mismatch.Action = "different-action"
	if _, err := repository.Create(ctx, actor, key, mismatch, "csv", true, audit); !errors.Is(err, ErrIdempotencyMismatch) {
		t.Fatalf("mismatch err=%v", err)
	}
	var ciphertext []byte
	if err := pool.QueryRow(ctx, `SELECT payload_cipher FROM audit_export_jobs WHERE id=$1`, job.ID).Scan(&ciphertext); err != nil {
		t.Fatalf("load ciphertext: %v", err)
	}
	if bytes.Contains(ciphertext, []byte("audit-canary-action")) {
		t.Fatal("export was stored in plaintext")
	}

	payload, format, err := repository.Consume(ctx, actor, job.ID, DownloadAudit{RequestID: "download-request", Method: "GET", RouteTemplate: "/v1/admin/audit-exports/{id}/download"})
	if err != nil {
		t.Fatalf("consume export: %v", err)
	}
	if format != "csv" || !bytes.Contains(payload, []byte("audit-canary-action")) {
		t.Fatalf("format=%s payload=%q", format, payload)
	}
	clear(payload)
	if _, _, err := repository.Consume(ctx, actor, job.ID, DownloadAudit{RequestID: "second", Method: "GET", RouteTemplate: "/v1/admin/audit-exports/{id}/download"}); !errors.Is(err, ErrExportConsumed) {
		t.Fatalf("second consume err=%v", err)
	}
	var accessEvents int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM admin_audit_events WHERE action='audit.export_downloaded' AND target_id=$1 AND request_id='download-request'`, job.ID.String()).Scan(&accessEvents); err != nil || accessEvents != 1 {
		t.Fatalf("access events=%d err=%v", accessEvents, err)
	}
}

func TestAsynchronousExportAndExpiry(t *testing.T) {
	pool := testsupport.NewDB(t)
	ctx := context.Background()
	now := time.Date(2026, 7, 11, 12, 0, 0, 0, time.UTC)
	repository := NewExportRepository(pool, exportCryptor(t), func() time.Time { return now })
	seedAuditExportEvent(t, pool, now.Add(-time.Hour), "async-action")
	actor := ExportActor{Kind: "shared_admin_token", Subject: "primary"}
	job, err := repository.Create(ctx, actor, uuid.New(), ExportFilter{From: now.Add(-24 * time.Hour), To: now}, "jsonl", false, DownloadAudit{RequestID: "create-async", Method: "POST", RouteTemplate: "/v1/admin/audit-exports"})
	if err != nil || job.Status != "pending" {
		t.Fatalf("pending job=%+v err=%v", job, err)
	}
	processed, err := repository.ProcessPending(ctx)
	if err != nil || !processed {
		t.Fatalf("processed=%v err=%v", processed, err)
	}
	ready, err := repository.Get(ctx, actor, job.ID)
	if err != nil || ready.Status != "ready" {
		t.Fatalf("ready=%+v err=%v", ready, err)
	}
	now = now.Add(25 * time.Hour)
	count, err := repository.Expire(ctx)
	if err != nil || count != 1 {
		t.Fatalf("expired=%d err=%v", count, err)
	}
	if _, _, err := repository.Consume(ctx, actor, job.ID, DownloadAudit{RequestID: "expired", Method: "GET", RouteTemplate: "/download"}); !errors.Is(err, ErrExportExpired) {
		t.Fatalf("expired consume err=%v", err)
	}
}

type failingArchiveSink struct{}

func (failingArchiveSink) Archive(context.Context, ArchiveMetadata, []byte) error {
	return errors.New("archive unavailable")
}

func TestArchiveSinkFailureIsClosedAndStoresNoPayload(t *testing.T) {
	pool := testsupport.NewDB(t)
	ctx := context.Background()
	now := time.Now().UTC()
	repository := NewExportRepository(pool, exportCryptor(t), func() time.Time { return now })
	repository.SetArchiveSink(failingArchiveSink{})
	actor := ExportActor{Kind: "shared_admin_token", Subject: "primary"}
	_, err := repository.Create(ctx, actor, uuid.New(), ExportFilter{From: now.Add(-time.Hour), To: now}, "csv", true, DownloadAudit{RequestID: "archive-failure", Method: "POST", RouteTemplate: "/v1/admin/audit-exports"})
	if err == nil {
		t.Fatal("archive failure was ignored")
	}
	var status string
	var hasPayload bool
	if scanErr := pool.QueryRow(ctx, `SELECT status,payload_cipher IS NOT NULL FROM audit_export_jobs WHERE requested_by_subject='primary'`).Scan(&status, &hasPayload); scanErr != nil {
		t.Fatal(scanErr)
	}
	if status != "failed" || hasPayload {
		t.Fatalf("status=%s payload=%v", status, hasPayload)
	}
}

func exportCryptor(t *testing.T) *appcrypto.Cryptor {
	t.Helper()
	cryptor, err := appcrypto.New(map[int][]byte{1: bytes.Repeat([]byte{1}, 32)}, 1)
	if err != nil {
		t.Fatal(err)
	}
	return cryptor
}

func seedAuditExportEvent(t *testing.T, pool *pgxpool.Pool, occurred time.Time, action string) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `INSERT INTO admin_audit_events
		(occurred_at,request_id,actor_kind,actor_subject,action,target_type,target_id,result,http_method,route_template,http_status,metadata)
		VALUES ($1,$2,'shared_admin_token','primary',$3,'test','target','success','POST','/test',200,'{}')`, occurred, uuid.NewString(), action); err != nil {
		t.Fatalf("seed audit event: %v", err)
	}
}

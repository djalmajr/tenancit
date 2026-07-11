package auditops

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/djalmajr/tenancit/server/internal/testsupport"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestMaintainCreatesFuturePartitionsIdempotently(t *testing.T) {
	pool := testsupport.NewDB(t)
	ctx := context.Background()
	now := time.Date(2026, 7, 11, 12, 0, 0, 0, time.UTC)
	if _, err := pool.Exec(ctx, `INSERT INTO admin_audit_events
		(occurred_at,request_id,actor_kind,actor_subject,action,target_type,target_id,result,http_method,route_template,http_status)
		VALUES ($1,'legacy-default','shared_admin_token','primary','test.legacy','test','legacy','success','POST','/test',200)`, now.Add(-time.Hour)); err != nil {
		t.Fatalf("insert legacy default event: %v", err)
	}

	first, err := Maintain(ctx, pool, now, 365, 3)
	if err != nil {
		t.Fatalf("first maintenance: %v", err)
	}
	if first.PartitionsCreated != 4 {
		t.Fatalf("created=%d want=4", first.PartitionsCreated)
	}
	second, err := Maintain(ctx, pool, now, 365, 3)
	if err != nil {
		t.Fatalf("second maintenance: %v", err)
	}
	if second.PartitionsCreated != 0 {
		t.Fatalf("idempotent created=%d want=0", second.PartitionsCreated)
	}
	var defaultRows int
	var legacyRelation string
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM admin_audit_events_default`).Scan(&defaultRows); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT tableoid::regclass::text FROM admin_audit_events WHERE request_id='legacy-default'`).Scan(&legacyRelation); err != nil {
		t.Fatal(err)
	}
	if defaultRows != 0 || legacyRelation != "admin_audit_events_202607" {
		t.Fatalf("default rows=%d relation=%s", defaultRows, legacyRelation)
	}
	if _, err := pool.Exec(ctx, `UPDATE admin_audit_events SET result='error' WHERE request_id='legacy-default'`); err == nil {
		t.Fatal("drained partition lost append-only trigger")
	}

	var routed string
	if err := pool.QueryRow(ctx, `INSERT INTO admin_audit_events
		(occurred_at,request_id,actor_kind,actor_subject,action,target_type,target_id,result,http_method,route_template,http_status)
		VALUES ('2026-08-10T12:00:00Z','partition-test','shared_admin_token','primary','test.partition','test','partition','success','POST','/test',200)
		RETURNING tableoid::regclass::text`).Scan(&routed); err != nil {
		t.Fatalf("insert routed event: %v", err)
	}
	if routed != "admin_audit_events_202608" {
		t.Fatalf("routed=%q", routed)
	}
	health, err := Health(ctx, pool, now)
	if err != nil {
		t.Fatalf("health: %v", err)
	}
	if !health.CurrentMonthCovered || !health.FutureThrough.Equal(time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("future through=%s", health.FutureThrough)
	}
}

func TestMaintainRetentionPreservesLegalHold(t *testing.T) {
	pool := testsupport.NewDB(t)
	ctx := context.Background()
	now := time.Date(2026, 7, 11, 12, 0, 0, 0, time.UTC)
	createTestPartition(t, pool, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
	createTestPartition(t, pool, time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC))
	if _, err := pool.Exec(ctx, `INSERT INTO audit_legal_holds
		(from_time,to_time,reason,created_by_kind,created_by_subject)
		VALUES ('2024-02-10','2024-02-20','incident-42','shared_admin_token','primary')`); err != nil {
		t.Fatalf("insert legal hold: %v", err)
	}

	result, err := Maintain(ctx, pool, now, 365, 1)
	if err != nil {
		t.Fatalf("maintenance: %v", err)
	}
	if result.PartitionsDropped != 1 || result.PartitionsHeld != 1 {
		t.Fatalf("result=%+v", result)
	}
	assertRelation(t, pool, "admin_audit_events_202401", false)
	assertRelation(t, pool, "admin_audit_events_202402", true)
}

func createTestPartition(t *testing.T, pool *pgxpool.Pool, from time.Time) {
	t.Helper()
	to := from.AddDate(0, 1, 0)
	name := partitionName(from)
	query := fmt.Sprintf(`CREATE TABLE %s PARTITION OF admin_audit_events FOR VALUES FROM ('%s') TO ('%s')`, pgx.Identifier{name}.Sanitize(), from.Format(time.RFC3339), to.Format(time.RFC3339))
	if _, err := pool.Exec(context.Background(), query); err != nil {
		t.Fatalf("create partition: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `INSERT INTO audit_partition_registry(partition_name,from_time,to_time) VALUES ($1,$2,$3)`, name, from, to); err != nil {
		t.Fatalf("register partition: %v", err)
	}
}

func assertRelation(t *testing.T, pool *pgxpool.Pool, name string, want bool) {
	t.Helper()
	var exists bool
	if err := pool.QueryRow(context.Background(), `SELECT to_regclass($1) IS NOT NULL`, name).Scan(&exists); err != nil {
		t.Fatalf("relation lookup: %v", err)
	}
	if exists != want {
		t.Fatalf("relation %s exists=%v want=%v", name, exists, want)
	}
}

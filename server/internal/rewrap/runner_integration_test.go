package rewrap

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"testing"
	"time"

	appcrypto "github.com/djalmajr/tenancit/server/internal/crypto"
	"github.com/djalmajr/tenancit/server/internal/testsupport"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestRewrapDryRunWriteResumeAndLogicalTimestamp(t *testing.T) {
	pool := testsupport.NewDB(t)
	oldCryptor, currentCryptor := testCryptors(t)
	resourceID := seedResource(t, pool, oldCryptor, []string{"alpha-secret", "beta-secret"})
	seedSafetyEvidence(t, pool)
	var before time.Time
	if err := pool.QueryRow(t.Context(), `SELECT updated_at FROM tenant_resources WHERE id=$1`, resourceID).Scan(&before); err != nil {
		t.Fatal(err)
	}
	beforeRows := encryptedSnapshot(t, pool)
	logBuffer := &bytes.Buffer{}
	reporter := &recordingReporter{}
	runner := &Runner{DB: pool, Cryptor: currentCryptor, Reporter: reporter, Logger: slog.New(slog.NewJSONHandler(logBuffer, nil))}
	config := Config{TargetVersion: 2, BatchSize: 1, DryRun: true, MaxDuration: time.Minute, JobID: uuid.NewString()}

	dry, err := runner.Run(t.Context(), config)
	if err != nil {
		t.Fatalf("dry-run: %v", err)
	}
	if dry.Scanned != 2 || dry.Rewrapped != 0 || dry.Remaining != 2 {
		t.Fatalf("dry summary = %+v", dry)
	}
	if !snapshotsEqual(beforeRows, encryptedSnapshot(t, pool)) {
		t.Fatal("dry-run changed encrypted bytes")
	}

	config.DryRun = false
	config.ConfirmedWrite = true
	config.JobID = uuid.NewString()
	written, err := runner.Run(t.Context(), config)
	if err != nil {
		t.Fatalf("write: %v", err)
	}
	if written.Rewrapped != 2 || written.Batches != 2 || written.Remaining != 0 || reporter.status != "healthy" {
		t.Fatalf("write summary=%+v report=%q", written, reporter.status)
	}
	assertSecrets(t, pool, currentCryptor, map[string]bool{"alpha-secret": true, "beta-secret": true}, 2)
	onlyCurrent, err := appcrypto.New(map[int][]byte{2: bytes.Repeat([]byte{0x22}, 32)}, 2)
	if err != nil {
		t.Fatal(err)
	}
	assertSecrets(t, pool, onlyCurrent, map[string]bool{"alpha-secret": true, "beta-secret": true}, 2)
	var after time.Time
	if err := pool.QueryRow(t.Context(), `SELECT updated_at FROM tenant_resources WHERE id=$1`, resourceID).Scan(&after); err != nil {
		t.Fatal(err)
	}
	if !before.Equal(after) {
		t.Fatalf("rewrap changed functional resource timestamp: before=%s after=%s", before, after)
	}

	config.JobID = uuid.NewString()
	rerun, err := runner.Run(t.Context(), config)
	if err != nil || rerun.Rewrapped != 0 || rerun.Remaining != 0 {
		t.Fatalf("idempotent rerun summary=%+v err=%v", rerun, err)
	}
	for _, canary := range []string{"alpha-secret", "beta-secret"} {
		if bytes.Contains(logBuffer.Bytes(), []byte(canary)) {
			t.Fatalf("log leaked plaintext canary %q", canary)
		}
	}
}

func TestRewrapPreflightFailuresWriteZeroRows(t *testing.T) {
	tests := []struct {
		name     string
		prepare  func(*testing.T, *pgxpool.Pool, *appcrypto.Cryptor, *appcrypto.Cryptor)
		cryptor  func(*appcrypto.Cryptor, *appcrypto.Cryptor) *appcrypto.Cryptor
		want     error
		evidence bool
	}{
		{name: "missing historical key", prepare: seedOneOldSecret, cryptor: func(_ *appcrypto.Cryptor, current *appcrypto.Cryptor) *appcrypto.Cryptor {
			onlyCurrent, _ := appcrypto.New(map[int][]byte{2: bytes.Repeat([]byte{0x22}, 32)}, 2)
			return onlyCurrent
		}, want: ErrMissingKeyVersion, evidence: true},
		{name: "tampered ciphertext", prepare: func(t *testing.T, db *pgxpool.Pool, old, _ *appcrypto.Cryptor) {
			seedOneOldSecret(t, db, old, nil)
			_, err := db.Exec(t.Context(), `UPDATE tenant_resource_values SET value_cipher=set_byte(value_cipher,0,get_byte(value_cipher,0)#255)`)
			if err != nil {
				t.Fatal(err)
			}
		}, cryptor: func(_ *appcrypto.Cryptor, current *appcrypto.Cryptor) *appcrypto.Cryptor { return current }, want: ErrAuthentication, evidence: true},
		{name: "malformed nonce", prepare: func(t *testing.T, db *pgxpool.Pool, old, _ *appcrypto.Cryptor) {
			seedOneOldSecret(t, db, old, nil)
			_, err := db.Exec(t.Context(), `UPDATE tenant_resource_values SET nonce='\\x01'::bytea`)
			if err != nil {
				t.Fatal(err)
			}
		}, cryptor: func(_ *appcrypto.Cryptor, current *appcrypto.Cryptor) *appcrypto.Cryptor { return current }, want: ErrMalformedCiphertext, evidence: true},
		{name: "safety evidence missing", prepare: seedOneOldSecret, cryptor: func(_ *appcrypto.Cryptor, current *appcrypto.Cryptor) *appcrypto.Cryptor { return current }, want: ErrSafetyEvidence},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			pool := testsupport.NewDB(t)
			oldCryptor, currentCryptor := testCryptors(t)
			test.prepare(t, pool, oldCryptor, currentCryptor)
			if test.evidence {
				seedSafetyEvidence(t, pool)
			}
			before := encryptedSnapshot(t, pool)
			runner := &Runner{DB: pool, Cryptor: test.cryptor(oldCryptor, currentCryptor)}
			_, err := runner.Run(t.Context(), Config{TargetVersion: 2, BatchSize: 10, ConfirmedWrite: true, MaxDuration: time.Minute, JobID: uuid.NewString()})
			if !errors.Is(err, test.want) {
				t.Fatalf("error=%v want %v", err, test.want)
			}
			if !snapshotsEqual(before, encryptedSnapshot(t, pool)) {
				t.Fatal("preflight failure changed rows")
			}
		})
	}
}

func TestRewrapBatchRollbackAndCampaignLock(t *testing.T) {
	pool := testsupport.NewDB(t)
	oldCryptor, currentCryptor := testCryptors(t)
	seedResource(t, pool, oldCryptor, []string{"first", "second"})
	seedSafetyEvidence(t, pool)
	before := encryptedSnapshot(t, pool)

	lock, err := pool.Acquire(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Release()
	if _, err := lock.Exec(t.Context(), `SELECT pg_advisory_lock($1)`, advisoryLockID); err != nil {
		t.Fatal(err)
	}
	runner := &Runner{DB: pool, Cryptor: currentCryptor}
	_, err = runner.Run(t.Context(), Config{TargetVersion: 2, BatchSize: 2, ConfirmedWrite: true, MaxDuration: time.Minute, JobID: uuid.NewString()})
	if !errors.Is(err, ErrCampaignLocked) {
		t.Fatalf("lock error=%v", err)
	}
	if _, err := lock.Exec(t.Context(), `SELECT pg_advisory_unlock($1)`, advisoryLockID); err != nil {
		t.Fatal(err)
	}

	var failID uuid.UUID
	if err := pool.QueryRow(t.Context(), `SELECT id FROM tenant_resource_values ORDER BY id DESC LIMIT 1`).Scan(&failID); err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(t.Context(), fmt.Sprintf(`CREATE FUNCTION fail_rewrap_test() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.id='%s' THEN RAISE EXCEPTION 'injected'; END IF; RETURN NEW; END $$;
		CREATE TRIGGER fail_rewrap_test BEFORE UPDATE ON tenant_resource_values FOR EACH ROW EXECUTE FUNCTION fail_rewrap_test()`, failID))
	if err != nil {
		t.Fatal(err)
	}
	_, err = runner.Run(t.Context(), Config{TargetVersion: 2, BatchSize: 2, ConfirmedWrite: true, MaxDuration: time.Minute, JobID: uuid.NewString()})
	if err == nil {
		t.Fatal("injected batch failure succeeded")
	}
	if !snapshotsEqual(before, encryptedSnapshot(t, pool)) {
		t.Fatal("failed batch was not rolled back")
	}
}

func TestRewrapSkipsConcurrentWriterAndTimesOutWhenLocked(t *testing.T) {
	pool := testsupport.NewDB(t)
	oldCryptor, currentCryptor := testCryptors(t)
	seedOneOldSecret(t, pool, oldCryptor, currentCryptor)
	seedSafetyEvidence(t, pool)
	var rowID uuid.UUID
	if err := pool.QueryRow(t.Context(), `SELECT id FROM tenant_resource_values LIMIT 1`).Scan(&rowID); err != nil {
		t.Fatal(err)
	}
	tx, err := pool.Begin(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(t.Context(), `SELECT 1 FROM tenant_resource_values WHERE id=$1 FOR UPDATE`, rowID); err != nil {
		t.Fatal(err)
	}

	runner := &Runner{DB: pool, Cryptor: currentCryptor}
	_, err = runner.Run(t.Context(), Config{TargetVersion: 2, BatchSize: 1, ConfirmedWrite: true, MaxDuration: time.Second, NoProgressTimeout: 100 * time.Millisecond, PollInterval: 10 * time.Millisecond, JobID: uuid.NewString()})
	if !errors.Is(err, ErrNoProgress) {
		t.Fatalf("locked row error=%v", err)
	}
	if err := tx.Rollback(t.Context()); err != nil {
		t.Fatal(err)
	}

	writer, err := pool.Begin(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer writer.Rollback(t.Context())
	newValue, err := currentCryptor.Encrypt("writer-new-value")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := writer.Exec(t.Context(), `UPDATE tenant_resource_values SET value_cipher=$1,nonce=$2,key_version=$3 WHERE id=$4`, newValue.Cipher, newValue.Nonce, newValue.KeyVersion, rowID); err != nil {
		t.Fatal(err)
	}
	type runResult struct {
		summary Summary
		err     error
	}
	result := make(chan runResult, 1)
	go func() {
		summary, runErr := runner.Run(context.Background(), Config{TargetVersion: 2, BatchSize: 1, ConfirmedWrite: true, MaxDuration: time.Second, NoProgressTimeout: 500 * time.Millisecond, PollInterval: 10 * time.Millisecond, JobID: uuid.NewString()})
		result <- runResult{summary: summary, err: runErr}
	}()
	time.Sleep(50 * time.Millisecond)
	if err := writer.Commit(t.Context()); err != nil {
		t.Fatal(err)
	}
	completed := <-result
	if completed.err != nil || completed.summary.Rewrapped != 0 || completed.summary.LockedRetries == 0 {
		t.Fatalf("concurrent writer summary=%+v err=%v", completed.summary, completed.err)
	}
	assertSecrets(t, pool, currentCryptor, map[string]bool{"writer-new-value": true}, 2)

	oldAgain, err := oldCryptor.Encrypt("cancel-preserves-old")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(t.Context(), `UPDATE tenant_resource_values SET value_cipher=$1,nonce=$2,key_version=1 WHERE id=$3`, oldAgain.Cipher, oldAgain.Nonce, rowID); err != nil {
		t.Fatal(err)
	}
	cancelWriter, err := pool.Begin(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer cancelWriter.Rollback(t.Context())
	if _, err := cancelWriter.Exec(t.Context(), `SELECT 1 FROM tenant_resource_values WHERE id=$1 FOR UPDATE`, rowID); err != nil {
		t.Fatal(err)
	}
	cancelContext, cancel := context.WithCancel(context.Background())
	go func() { time.Sleep(30 * time.Millisecond); cancel() }()
	_, err = runner.Run(cancelContext, Config{TargetVersion: 2, BatchSize: 1, ConfirmedWrite: true, MaxDuration: time.Second, NoProgressTimeout: time.Second, PollInterval: 10 * time.Millisecond, JobID: uuid.NewString()})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel error=%v", err)
	}
	var preservedVersion int
	if err := cancelWriter.QueryRow(t.Context(), `SELECT key_version FROM tenant_resource_values WHERE id=$1`, rowID).Scan(&preservedVersion); err != nil {
		t.Fatal(err)
	}
	if preservedVersion != 1 {
		t.Fatal("canceled campaign changed locked row")
	}
}

func TestRewrapRestoredSnapshotDrill(t *testing.T) {
	adminURL := os.Getenv("TENANCIT_TEST_DATABASE_URL")
	if adminURL == "" {
		t.Skip("restore snapshot drill requires shared PostgreSQL admin DSN")
	}
	source := testsupport.NewDB(t)
	oldCryptor, currentCryptor := testCryptors(t)
	seedResource(t, source, oldCryptor, []string{"restored-snapshot-canary"})
	seedSafetyEvidence(t, source)
	var sourceDatabase string
	if err := source.QueryRow(t.Context(), `SELECT current_database()`).Scan(&sourceDatabase); err != nil {
		t.Fatal(err)
	}
	source.Close()

	admin, err := pgx.Connect(t.Context(), adminURL)
	if err != nil {
		t.Fatal(err)
	}
	restoreDatabase := "tenancit_rewrap_restore_" + uuid.NewString()[:12]
	t.Cleanup(func() {
		_, _ = admin.Exec(context.Background(), `SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname=$1 AND pid<>pg_backend_pid()`, restoreDatabase)
		_, _ = admin.Exec(context.Background(), "DROP DATABASE IF EXISTS "+pgx.Identifier{restoreDatabase}.Sanitize())
		_ = admin.Close(context.Background())
	})
	if _, err := admin.Exec(t.Context(), `SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname=$1 AND pid<>pg_backend_pid()`, sourceDatabase); err != nil {
		t.Fatal(err)
	}
	if _, err := admin.Exec(t.Context(), "CREATE DATABASE "+pgx.Identifier{restoreDatabase}.Sanitize()+" TEMPLATE "+pgx.Identifier{sourceDatabase}.Sanitize()); err != nil {
		t.Fatal(err)
	}
	restoredURL, err := url.Parse(adminURL)
	if err != nil {
		t.Fatal(err)
	}
	restoredURL.Path = "/" + restoreDatabase
	restored, err := pgxpool.New(t.Context(), restoredURL.String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(restored.Close)
	runner := &Runner{DB: restored, Cryptor: currentCryptor, Reporter: &recordingReporter{}}
	summary, err := runner.Run(t.Context(), Config{TargetVersion: 2, BatchSize: 10, ConfirmedWrite: true, MaxDuration: time.Minute, JobID: uuid.NewString()})
	if err != nil || summary.Rewrapped != 1 || summary.Remaining != 0 {
		t.Fatalf("restored drill summary=%+v err=%v", summary, err)
	}
	assertSecrets(t, restored, currentCryptor, map[string]bool{"restored-snapshot-canary": true}, 2)
}

type recordingReporter struct{ status string }

func (r *recordingReporter) Report(_ context.Context, _ Summary, status string) error {
	r.status = status
	return nil
}

type rowSnapshot struct {
	ID            uuid.UUID
	Cipher, Nonce []byte
	Version       *int32
}

func testCryptors(t *testing.T) (*appcrypto.Cryptor, *appcrypto.Cryptor) {
	t.Helper()
	key1 := bytes.Repeat([]byte{0x11}, 32)
	key2 := bytes.Repeat([]byte{0x22}, 32)
	oldCryptor, err := appcrypto.New(map[int][]byte{1: key1}, 1)
	if err != nil {
		t.Fatal(err)
	}
	currentCryptor, err := appcrypto.New(map[int][]byte{1: key1, 2: key2}, 2)
	if err != nil {
		t.Fatal(err)
	}
	return oldCryptor, currentCryptor
}

func seedOneOldSecret(t *testing.T, pool *pgxpool.Pool, old, _ *appcrypto.Cryptor) {
	t.Helper()
	seedResource(t, pool, old, []string{"preflight-canary"})
}

func seedResource(t *testing.T, pool *pgxpool.Pool, cryptor *appcrypto.Cryptor, secrets []string) uuid.UUID {
	t.Helper()
	var resourceID uuid.UUID
	err := pool.QueryRow(t.Context(), `WITH tenant AS (INSERT INTO tenants(slug,name) VALUES($1,$1) RETURNING id),
		definition AS (INSERT INTO resource_definitions(key,name) VALUES($2,$2) RETURNING id)
		INSERT INTO tenant_resources(tenant_id,resource_definition_id)
		SELECT tenant.id,definition.id FROM tenant,definition RETURNING id`, "tenant-"+uuid.NewString(), "definition-"+uuid.NewString()).Scan(&resourceID)
	if err != nil {
		t.Fatal(err)
	}
	for index, secret := range secrets {
		var fieldID uuid.UUID
		if err := pool.QueryRow(t.Context(), `INSERT INTO resource_fields(resource_definition_id,key,label,is_secret,sort_order)
			SELECT resource_definition_id,$2,$2,true,$3 FROM tenant_resources WHERE id=$1 RETURNING id`, resourceID, fmt.Sprintf("secret-%d", index), index).Scan(&fieldID); err != nil {
			t.Fatal(err)
		}
		encrypted, err := cryptor.Encrypt(secret)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(t.Context(), `INSERT INTO tenant_resource_values(tenant_resource_id,resource_field_id,value_cipher,nonce,key_version) VALUES($1,$2,$3,$4,$5)`, resourceID, fieldID, encrypted.Cipher, encrypted.Nonce, encrypted.KeyVersion); err != nil {
			t.Fatal(err)
		}
	}
	return resourceID
}

func seedSafetyEvidence(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	for _, kind := range []string{"backup", "restore"} {
		_, err := pool.Exec(t.Context(), `INSERT INTO operational_reports(kind,source,status,occurred_at,fresh_until,idempotency_key,payload_hash,credential_version)
			VALUES($1,'rewrap-test','healthy',clock_timestamp(),clock_timestamp()+interval '1 hour',$1||'-'||gen_random_uuid(),decode('00','hex'),'test-v1')`, kind)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func encryptedSnapshot(t *testing.T, pool *pgxpool.Pool) []rowSnapshot {
	t.Helper()
	rows, err := pool.Query(t.Context(), `SELECT id,value_cipher,nonce,key_version FROM tenant_resource_values ORDER BY id`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var snapshots []rowSnapshot
	for rows.Next() {
		var item rowSnapshot
		if err := rows.Scan(&item.ID, &item.Cipher, &item.Nonce, &item.Version); err != nil {
			t.Fatal(err)
		}
		item.Cipher = bytes.Clone(item.Cipher)
		item.Nonce = bytes.Clone(item.Nonce)
		snapshots = append(snapshots, item)
	}
	return snapshots
}

func snapshotsEqual(a, b []rowSnapshot) bool {
	if len(a) != len(b) {
		return false
	}
	for index := range a {
		if a[index].ID != b[index].ID || !bytes.Equal(a[index].Cipher, b[index].Cipher) || !bytes.Equal(a[index].Nonce, b[index].Nonce) {
			return false
		}
		if (a[index].Version == nil) != (b[index].Version == nil) || (a[index].Version != nil && *a[index].Version != *b[index].Version) {
			return false
		}
	}
	return true
}

func assertSecrets(t *testing.T, pool *pgxpool.Pool, cryptor *appcrypto.Cryptor, expected map[string]bool, version int) {
	t.Helper()
	rows, err := pool.Query(t.Context(), `SELECT value_cipher,nonce,key_version FROM tenant_resource_values WHERE value_cipher IS NOT NULL`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	found := map[string]bool{}
	for rows.Next() {
		var encrypted appcrypto.Encrypted
		if err := rows.Scan(&encrypted.Cipher, &encrypted.Nonce, &encrypted.KeyVersion); err != nil {
			t.Fatal(err)
		}
		if encrypted.KeyVersion != version {
			t.Fatalf("key version=%d", encrypted.KeyVersion)
		}
		plaintext, err := cryptor.Decrypt(encrypted)
		if err != nil {
			t.Fatal(err)
		}
		found[plaintext] = true
	}
	for value := range expected {
		if !found[value] {
			t.Fatalf("missing plaintext %q after rewrap", value)
		}
	}
}

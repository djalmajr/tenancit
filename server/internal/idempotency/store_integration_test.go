package idempotency

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	appcrypto "github.com/djalmajr/tenancit/server/internal/crypto"
	"github.com/djalmajr/tenancit/server/internal/testsupport"
	"github.com/google/uuid"
)

func TestReplayMismatchAndEncryptedStorage(t *testing.T) {
	pool := testsupport.NewDB(t)
	now := time.Date(2026, 7, 11, 12, 0, 0, 0, time.UTC)
	store := testStore(t, now)
	request := testRequest(uuid.New(), Fingerprint("create", []byte(`{"name":"first"}`)))

	tx, err := pool.Begin(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if replay, err := store.Begin(context.Background(), tx, request); err != nil || replay.Found {
		t.Fatalf("initial begin replay=%+v err=%v", replay, err)
	}
	secretResponse := []byte(`{"token":"secret-once"}`)
	if err := store.Complete(context.Background(), tx, request, 201, "application/json", secretResponse); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(context.Background()); err != nil {
		t.Fatal(err)
	}

	var stored []byte
	if err := pool.QueryRow(context.Background(), `SELECT response_cipher FROM admin_idempotency_records WHERE idempotency_key=$1`, request.Key).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(stored, []byte("secret-once")) {
		t.Fatal("plaintext secret persisted in idempotency record")
	}

	tx, _ = pool.Begin(context.Background())
	replay, err := store.Begin(context.Background(), tx, request)
	if err != nil || !replay.Found || replay.Status != 201 || !bytes.Equal(replay.Body, secretResponse) {
		t.Fatalf("replay=%+v err=%v", replay, err)
	}
	clear(replay.Body)
	_ = tx.Rollback(context.Background())

	mismatch := request
	mismatch.Fingerprint = Fingerprint("create", []byte(`{"name":"different"}`))
	tx, _ = pool.Begin(context.Background())
	_, err = store.Begin(context.Background(), tx, mismatch)
	_ = tx.Rollback(context.Background())
	if !errors.Is(err, ErrFingerprintMismatch) {
		t.Fatalf("mismatch err=%v", err)
	}
}

func TestConcurrentRetryWaitsForSingleCommittedResult(t *testing.T) {
	pool := testsupport.NewDB(t)
	store := testStore(t, time.Now().UTC())
	request := testRequest(uuid.New(), Fingerprint("rotate", []byte(`{"grace":300}`)))
	ctx := context.Background()
	tx, _ := pool.Begin(ctx)
	if _, err := store.Begin(ctx, tx, request); err != nil {
		t.Fatal(err)
	}

	type result struct {
		replay Replay
		err    error
	}
	done := make(chan result, 1)
	go func() {
		retryTx, _ := pool.Begin(ctx)
		replay, err := store.Begin(ctx, retryTx, request)
		_ = retryTx.Rollback(ctx)
		done <- result{replay: replay, err: err}
	}()
	select {
	case <-done:
		t.Fatal("concurrent retry did not wait for owner transaction")
	case <-time.After(50 * time.Millisecond):
	}
	body := []byte(`{"token":"same-successor"}`)
	if err := store.Complete(ctx, tx, request, 201, "application/json", body); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case outcome := <-done:
		if outcome.err != nil || !outcome.replay.Found || !bytes.Equal(outcome.replay.Body, body) {
			t.Fatalf("retry=%+v err=%v", outcome.replay, outcome.err)
		}
		clear(outcome.replay.Body)
	case <-time.After(2 * time.Second):
		t.Fatal("concurrent retry remained blocked")
	}
	var records int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM admin_idempotency_records`).Scan(&records); err != nil || records != 1 {
		t.Fatalf("records=%d err=%v", records, err)
	}
}

func TestRollbackReleasesClaimAndCleanupRemovesExpired(t *testing.T) {
	pool := testsupport.NewDB(t)
	now := time.Date(2026, 7, 11, 12, 0, 0, 0, time.UTC)
	store := testStore(t, now)
	request := testRequest(uuid.New(), Fingerprint("provision", []byte(`{}`)))
	ctx := context.Background()
	tx, _ := pool.Begin(ctx)
	if _, err := store.Begin(ctx, tx, request); err != nil {
		t.Fatal(err)
	}
	_ = tx.Rollback(ctx)

	tx, _ = pool.Begin(ctx)
	if replay, err := store.Begin(ctx, tx, request); err != nil || replay.Found {
		t.Fatalf("begin after rollback replay=%+v err=%v", replay, err)
	}
	if err := store.Complete(ctx, tx, request, 201, "application/json", []byte(`{"id":"one"}`)); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}

	tx, _ = pool.Begin(ctx)
	removed, err := CleanupExpired(ctx, tx, now.Add(25*time.Hour), 1000)
	if err != nil || removed != 1 {
		t.Fatalf("removed=%d err=%v", removed, err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
}

func testStore(t *testing.T, now time.Time) *Store {
	t.Helper()
	key := bytes.Repeat([]byte{7}, 32)
	cryptor, err := appcrypto.New(map[int][]byte{1: key}, 1)
	if err != nil {
		t.Fatal(err)
	}
	return NewStore(cryptor, func() time.Time { return now })
}

func testRequest(key uuid.UUID, fingerprint [32]byte) Request {
	return Request{
		Actor: Actor{Kind: "shared_admin_token", Subject: "primary"}, Operation: "create",
		Key: key, Fingerprint: fingerprint, TTL: 24 * time.Hour,
	}
}

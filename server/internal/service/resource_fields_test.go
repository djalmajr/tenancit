package service

import (
	"context"
	"testing"

	"github.com/djalmajr/tenancit/server/internal/store/db"
	"github.com/google/uuid"
)

type fakeResourceBatchQuerier struct {
	calls int
	rows  []db.ListResourceFieldValuesByResourceIDsRow
}

func (f *fakeResourceBatchQuerier) ListResourceFieldValuesByResourceIDs(
	_ context.Context,
	_ []uuid.UUID,
) ([]db.ListResourceFieldValuesByResourceIDsRow, error) {
	f.calls++
	return f.rows, nil
}

func TestBuildResourceFieldsBatchUsesOneQueryAndPreservesEmptyResources(t *testing.T) {
	cryptor := newCryptor(t)
	resourceA := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	resourceB := uuid.MustParse("00000000-0000-4000-8000-000000000002")
	hostField := uuid.MustParse("00000000-0000-4000-8000-000000000011")
	passwordField := uuid.MustParse("00000000-0000-4000-8000-000000000012")
	host := "db.internal"
	secret, err := encodeValue(cryptor, true, "hunter2")
	if err != nil {
		t.Fatalf("encode secret: %v", err)
	}
	fake := &fakeResourceBatchQuerier{rows: []db.ListResourceFieldValuesByResourceIDsRow{
		{
			TenantResourceID: resourceA, ResourceFieldID: hostField, FieldKey: "host",
			DataType: "string", Required: true, HasValue: true, ValuePlain: &host,
		},
		{
			TenantResourceID: resourceA, ResourceFieldID: passwordField, FieldKey: "password",
			DataType: "string", Required: true, IsSecret: true, HasValue: true,
			ValueCipher: secret.ValueCipher, Nonce: secret.Nonce, KeyVersion: secret.KeyVersion,
		},
	}}
	headers := []ResourceHeader{
		{Resource: db.TenantResource{ID: resourceA}, DefinitionKey: "postgres"},
		{Resource: db.TenantResource{ID: resourceB}, DefinitionKey: "empty"},
	}

	masked, err := BuildResourceFieldsBatch(context.Background(), fake, cryptor, headers, false)
	if err != nil {
		t.Fatalf("BuildResourceFieldsBatch: %v", err)
	}
	if fake.calls != 1 {
		t.Fatalf("batch query calls = %d, want 1", fake.calls)
	}
	if len(masked) != 2 || len(masked[0].Fields) != 2 || len(masked[1].Fields) != 0 {
		t.Fatalf("unexpected batch shape: %+v", masked)
	}
	if masked[0].Fields[0].Value != host || masked[0].Fields[1].Value != MaskedValue {
		t.Fatalf("mask behavior changed: %+v", masked[0].Fields)
	}

	revealed, err := BuildResourceFieldsBatch(context.Background(), fake, cryptor, headers, true)
	if err != nil {
		t.Fatalf("BuildResourceFieldsBatch reveal: %v", err)
	}
	if revealed[0].Fields[1].Value != "hunter2" {
		t.Fatalf("secret not revealed: %+v", revealed[0].Fields)
	}
}

func TestBuildResourceFieldsBatchSkipsQueryForNoResources(t *testing.T) {
	fake := &fakeResourceBatchQuerier{}
	got, err := BuildResourceFieldsBatch(context.Background(), fake, newCryptor(t), nil, true)
	if err != nil {
		t.Fatalf("BuildResourceFieldsBatch: %v", err)
	}
	if fake.calls != 0 || len(got) != 0 {
		t.Fatalf("empty batch queried=%d len=%d", fake.calls, len(got))
	}
}

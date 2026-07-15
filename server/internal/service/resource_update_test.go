package service

import (
	"context"
	"errors"
	"testing"

	"github.com/djalmajr/tenancit/server/internal/store/db"
	"github.com/google/uuid"
)

type fakeResourceUpdateQuerier struct {
	fields   []db.ResourceField
	resource db.TenantResource
	upsert   db.UpsertResourceValueParams
}

func (f *fakeResourceUpdateQuerier) GetTenantResource(context.Context, db.GetTenantResourceParams) (db.TenantResource, error) {
	return f.resource, nil
}

func (f *fakeResourceUpdateQuerier) ListFields(context.Context, uuid.UUID) ([]db.ResourceField, error) {
	return f.fields, nil
}

func (f *fakeResourceUpdateQuerier) UpsertResourceValue(_ context.Context, arg db.UpsertResourceValueParams) (db.UpsertResourceValueRow, error) {
	f.upsert = arg
	return db.UpsertResourceValueRow{}, nil
}

func TestUpdateResourceFieldRejectsUnknownAndEmptyRequiredFields(t *testing.T) {
	definitionID := uuid.New()
	resourceID := uuid.New()
	fake := &fakeResourceUpdateQuerier{
		resource: db.TenantResource{ID: resourceID, ResourceDefinitionID: definitionID},
		fields:   []db.ResourceField{{ID: uuid.New(), Key: "host", Required: true}},
	}

	_, err := UpdateResourceFieldInTx(context.Background(), fake, newCryptor(t), UpdateResourceFieldInput{FieldKey: "port", ResourceID: resourceID})
	if !errors.Is(err, ErrUnknownResourceField) {
		t.Fatalf("unknown field error = %v", err)
	}
	_, err = UpdateResourceFieldInTx(context.Background(), fake, newCryptor(t), UpdateResourceFieldInput{FieldKey: "host", ResourceID: resourceID})
	var missing MissingRequiredFieldError
	if !errors.As(err, &missing) || missing.Key != "host" {
		t.Fatalf("missing required field error = %v", err)
	}
}

func TestUpdateResourceFieldEncryptsSecretValue(t *testing.T) {
	definitionID := uuid.New()
	resourceID := uuid.New()
	fieldID := uuid.New()
	fake := &fakeResourceUpdateQuerier{
		resource: db.TenantResource{ID: resourceID, ResourceDefinitionID: definitionID},
		fields:   []db.ResourceField{{ID: fieldID, Key: "password", IsSecret: true}},
	}

	field, err := UpdateResourceFieldInTx(context.Background(), fake, newCryptor(t), UpdateResourceFieldInput{
		FieldKey: "password", ResourceID: resourceID, Value: "hunter2",
	})
	if err != nil {
		t.Fatalf("UpdateResourceFieldInTx: %v", err)
	}
	if field.ID != fieldID || fake.upsert.ValuePlain != nil || len(fake.upsert.ValueCipher) == 0 {
		t.Fatalf("secret update was not encrypted: field=%+v upsert=%+v", field, fake.upsert)
	}
}

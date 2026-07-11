package service

import (
	"context"
	"errors"
	"testing"

	"github.com/djalmajr/tenancit/server/internal/store/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type fakeProvisionQuerier struct {
	definition db.ResourceDefinition
	defErr     error
	fields     []db.ResourceField
	resource   db.TenantResource
	upserts    []db.UpsertResourceValueParams
}

func (f *fakeProvisionQuerier) GetDefinitionByKey(context.Context, string) (db.ResourceDefinition, error) {
	return f.definition, f.defErr
}

func (f *fakeProvisionQuerier) ListFields(context.Context, uuid.UUID) ([]db.ResourceField, error) {
	return f.fields, nil
}

func (f *fakeProvisionQuerier) CreateTenantResource(context.Context, db.CreateTenantResourceParams) (db.TenantResource, error) {
	return f.resource, nil
}

func (f *fakeProvisionQuerier) UpsertResourceValue(_ context.Context, arg db.UpsertResourceValueParams) (db.UpsertResourceValueRow, error) {
	f.upserts = append(f.upserts, arg)
	return db.UpsertResourceValueRow{}, nil
}

func TestProvisionResourceRejectsUnknownAndInactiveDefinitions(t *testing.T) {
	_, err := provisionResource(context.Background(), &fakeProvisionQuerier{defErr: pgx.ErrNoRows}, newCryptor(t), ProvisionResourceInput{})
	if !errors.Is(err, ErrUnknownDefinition) {
		t.Fatalf("unknown definition error = %v", err)
	}

	_, err = provisionResource(context.Background(), &fakeProvisionQuerier{
		definition: db.ResourceDefinition{Status: "inactive"},
	}, newCryptor(t), ProvisionResourceInput{})
	if !errors.Is(err, ErrInactiveDefinition) {
		t.Fatalf("inactive definition error = %v", err)
	}
}

func TestProvisionResourceRejectsMissingRequiredField(t *testing.T) {
	definitionID := uuid.New()
	_, err := provisionResource(context.Background(), &fakeProvisionQuerier{
		definition: db.ResourceDefinition{ID: definitionID, Status: "active"},
		fields: []db.ResourceField{
			{ID: uuid.New(), ResourceDefinitionID: definitionID, Key: "password", Required: true},
		},
	}, newCryptor(t), ProvisionResourceInput{Values: map[string]string{}})
	var missing MissingRequiredFieldError
	if !errors.As(err, &missing) || missing.Key != "password" {
		t.Fatalf("missing required field error = %v", err)
	}
}

func TestProvisionResourceEncryptsSecretValues(t *testing.T) {
	definitionID := uuid.New()
	resourceID := uuid.New()
	hostFieldID := uuid.New()
	passwordFieldID := uuid.New()
	fake := &fakeProvisionQuerier{
		definition: db.ResourceDefinition{ID: definitionID, Status: "active"},
		resource:   db.TenantResource{ID: resourceID},
		fields: []db.ResourceField{
			{ID: hostFieldID, ResourceDefinitionID: definitionID, Key: "host", Required: true},
			{ID: passwordFieldID, ResourceDefinitionID: definitionID, Key: "password", Required: true, IsSecret: true},
		},
	}

	_, err := provisionResource(context.Background(), fake, newCryptor(t), ProvisionResourceInput{
		Values: map[string]string{"host": "db.internal", "password": "hunter2"},
	})
	if err != nil {
		t.Fatalf("provisionResource: %v", err)
	}
	if len(fake.upserts) != 2 {
		t.Fatalf("upserts = %d, want 2", len(fake.upserts))
	}
	if fake.upserts[0].ValuePlain == nil || *fake.upserts[0].ValuePlain != "db.internal" {
		t.Fatalf("host was not stored as plain value: %+v", fake.upserts[0])
	}
	secret := fake.upserts[1]
	if secret.ValuePlain != nil || len(secret.ValueCipher) == 0 || len(secret.Nonce) == 0 || secret.KeyVersion == nil {
		t.Fatalf("secret was not encrypted: %+v", secret)
	}
}

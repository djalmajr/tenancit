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
	linkSource db.TenantResource
	linkValues []db.TenantResourceValue
	created    db.CreateTenantResourceParams
	upserts    []db.UpsertResourceValueParams
}

func (f *fakeProvisionQuerier) GetResourceForLink(context.Context, db.GetResourceForLinkParams) (db.TenantResource, error) {
	if f.linkSource.ID == uuid.Nil {
		return db.TenantResource{}, pgx.ErrNoRows
	}
	return f.linkSource, nil
}

func (f *fakeProvisionQuerier) ListResourceValues(context.Context, uuid.UUID) ([]db.TenantResourceValue, error) {
	return f.linkValues, nil
}

func (f *fakeProvisionQuerier) GetDefinitionByKey(context.Context, string) (db.ResourceDefinition, error) {
	return f.definition, f.defErr
}

func (f *fakeProvisionQuerier) ListFields(context.Context, uuid.UUID) ([]db.ResourceField, error) {
	return f.fields, nil
}

func (f *fakeProvisionQuerier) CreateTenantResource(_ context.Context, arg db.CreateTenantResourceParams) (db.TenantResource, error) {
	f.created = arg
	return f.resource, nil
}

func TestProvisionResourceLinksToSourceAndStoresOnlyOverrides(t *testing.T) {
	tenantID := uuid.New()
	definitionID := uuid.New()
	hostFieldID := uuid.New()
	sourceID := uuid.New()
	fake := &fakeProvisionQuerier{
		definition: db.ResourceDefinition{ID: definitionID, Key: "postgres", Status: "active"},
		fields: []db.ResourceField{
			{ID: hostFieldID, ResourceDefinitionID: definitionID, Key: "host", Required: true},
			{ID: uuid.New(), ResourceDefinitionID: definitionID, Key: "schema", Required: true},
		},
		linkSource: db.TenantResource{ID: sourceID, TenantID: tenantID, ResourceDefinitionID: definitionID, Status: "active"},
		linkValues: []db.TenantResourceValue{{TenantResourceID: sourceID, ResourceFieldID: hostFieldID}},
		resource:   db.TenantResource{ID: uuid.New()},
	}

	_, err := provisionResource(context.Background(), fake, newCryptor(t), ProvisionResourceInput{
		Alias: "postgres.agility", DefinitionKey: "postgres", SourceResourceID: &sourceID,
		TenantID: tenantID, Values: map[string]string{"schema": "agility"},
	})
	if err != nil {
		t.Fatalf("provision linked resource: %v", err)
	}
	if !fake.created.SourceResourceID.Valid || fake.created.SourceResourceID.Bytes != sourceID {
		t.Fatalf("source id = %+v", fake.created.SourceResourceID)
	}
	if len(fake.upserts) != 1 || fake.upserts[0].ResourceFieldID == hostFieldID {
		t.Fatalf("upserts = %+v, want only local schema override", fake.upserts)
	}
}

type fakeDuplicateQuerier struct {
	fakeProvisionQuerier
	header db.GetResourceHeaderRow
	rows   []db.ListResourceFieldValuesByResourceIDsRow
}

func (f *fakeDuplicateQuerier) GetResourceHeader(context.Context, uuid.UUID) (db.GetResourceHeaderRow, error) {
	return f.header, nil
}

func (f *fakeDuplicateQuerier) ListResourceFieldValuesByResourceIDs(context.Context, []uuid.UUID) ([]db.ListResourceFieldValuesByResourceIDsRow, error) {
	return f.rows, nil
}

func TestDuplicateResourceCreatesIndependentEffectiveSnapshot(t *testing.T) {
	tenantID := uuid.New()
	definitionID := uuid.New()
	resourceID := uuid.New()
	fieldID := uuid.New()
	value := "shared.internal"
	fake := &fakeDuplicateQuerier{
		fakeProvisionQuerier: fakeProvisionQuerier{
			definition: db.ResourceDefinition{ID: definitionID, Key: "postgres", Status: "active"},
			fields:     []db.ResourceField{{ID: fieldID, ResourceDefinitionID: definitionID, Key: "host", Required: true}},
			resource:   db.TenantResource{ID: uuid.New()},
		},
		header: db.GetResourceHeaderRow{
			ID: resourceID, TenantID: tenantID, ResourceDefinitionID: definitionID,
			Alias: "postgres.agility", DefinitionKey: "postgres", DefinitionName: "Postgres",
		},
		rows: []db.ListResourceFieldValuesByResourceIDsRow{{
			TenantResourceID: resourceID, ResourceFieldID: fieldID, FieldKey: "host",
			DataType: "string", Required: true, HasValue: true, ValuePlain: &value,
		}},
	}

	_, err := DuplicateResourceInTx(context.Background(), fake, newCryptor(t), tenantID, resourceID, "postgres.copy")
	if err != nil {
		t.Fatalf("duplicate resource: %v", err)
	}
	if fake.created.SourceResourceID.Valid {
		t.Fatalf("duplicate unexpectedly linked to source: %+v", fake.created.SourceResourceID)
	}
	if fake.created.Alias != "postgres.copy" || len(fake.upserts) != 1 || fake.upserts[0].ValuePlain == nil || *fake.upserts[0].ValuePlain != value {
		t.Fatalf("snapshot = alias %q, values %+v", fake.created.Alias, fake.upserts)
	}
}

func (f *fakeProvisionQuerier) UpsertResourceValue(_ context.Context, arg db.UpsertResourceValueParams) (db.UpsertResourceValueRow, error) {
	f.upserts = append(f.upserts, arg)
	return db.UpsertResourceValueRow{}, nil
}

func TestProvisionResourceRejectsUnknownAndInactiveDefinitions(t *testing.T) {
	_, err := provisionResource(context.Background(), &fakeProvisionQuerier{defErr: pgx.ErrNoRows}, newCryptor(t), ProvisionResourceInput{DefinitionKey: "postgres"})
	if !errors.Is(err, ErrUnknownDefinition) {
		t.Fatalf("unknown definition error = %v", err)
	}

	_, err = provisionResource(context.Background(), &fakeProvisionQuerier{
		definition: db.ResourceDefinition{Status: "inactive"},
	}, newCryptor(t), ProvisionResourceInput{DefinitionKey: "postgres"})
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
	}, newCryptor(t), ProvisionResourceInput{DefinitionKey: "postgres", Values: map[string]string{}})
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
		DefinitionKey: "postgres",
		Values:        map[string]string{"host": "db.internal", "password": "hunter2"},
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

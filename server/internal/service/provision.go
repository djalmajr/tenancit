package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/djalmajr/tenancit/server/internal/crypto"
	"github.com/djalmajr/tenancit/server/internal/store"
	"github.com/djalmajr/tenancit/server/internal/store/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

var (
	ErrResourceAliasExists   = errors.New("resource alias already exists")
	ErrInvalidResourceAlias  = errors.New("invalid resource alias")
	ErrInvalidResourceName   = errors.New("invalid resource name")
	ErrInvalidResourceSource = errors.New("invalid resource source")
	ErrInactiveDefinition    = errors.New("definition is inactive")
	ErrUnknownDefinition     = errors.New("unknown definition")
	ErrUnknownTenant         = errors.New("unknown tenant")
)

type MissingRequiredFieldError struct {
	Key string
}

type InvalidFieldValueError struct {
	Key string
}

func (e InvalidFieldValueError) Error() string {
	return "invalid value for field: " + e.Key
}

func (e MissingRequiredFieldError) Error() string {
	return "missing required field: " + e.Key
}

type ProvisionResourceInput struct {
	Name             string
	Alias            string
	DefinitionKey    string
	SourceResourceID *uuid.UUID
	TenantID         uuid.UUID
	Values           map[string]string
}

type ProvisionResourceDeps struct {
	Cryptor      *crypto.Cryptor
	Queries      *db.Queries
	TxStarter    ResourceTransactor
	BeforeCommit func(*db.Queries, db.TenantResource) error
}

type ResourceTransactor interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

type ProvisionQuerier interface {
	GetDefinitionByKey(ctx context.Context, key string) (db.ResourceDefinition, error)
	ListFields(ctx context.Context, resourceDefinitionID uuid.UUID) ([]db.ResourceField, error)
	CreateTenantResource(ctx context.Context, arg db.CreateTenantResourceParams) (db.TenantResource, error)
	GetResourceForLink(ctx context.Context, arg db.GetResourceForLinkParams) (db.TenantResource, error)
	ListResourceValues(ctx context.Context, tenantResourceID uuid.UUID) ([]db.TenantResourceValue, error)
	UpsertResourceValue(ctx context.Context, arg db.UpsertResourceValueParams) (db.UpsertResourceValueRow, error)
}

type DuplicateResourceQuerier interface {
	ProvisionQuerier
	ResourceBatchQuerier
	GetResourceHeader(ctx context.Context, id uuid.UUID) (db.GetResourceHeaderRow, error)
}

func ProvisionResource(ctx context.Context, deps ProvisionResourceDeps, in ProvisionResourceInput) (db.TenantResource, error) {
	tx, err := deps.TxStarter.Begin(ctx)
	if err != nil {
		return db.TenantResource{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	res, err := ProvisionResourceInTx(ctx, deps.Queries.WithTx(tx), deps.Cryptor, in)
	if err != nil {
		return db.TenantResource{}, err
	}
	if deps.BeforeCommit != nil {
		if err := deps.BeforeCommit(deps.Queries.WithTx(tx), res); err != nil {
			return db.TenantResource{}, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return db.TenantResource{}, err
	}
	return res, nil
}

// ProvisionResourceInTx applies provisioning inside a caller-owned transaction.
// It lets HTTP orchestration commit domain, audit, outbox, and idempotency as one unit.
func ProvisionResourceInTx(ctx context.Context, q ProvisionQuerier, cryptor *crypto.Cryptor, in ProvisionResourceInput) (db.TenantResource, error) {
	return provisionResource(ctx, q, cryptor, in)
}

// DuplicateResourceInTx creates an independent snapshot of a resource's
// effective values. A duplicate never keeps a link to the original resource.
func DuplicateResourceInTx(
	ctx context.Context,
	q DuplicateResourceQuerier,
	cryptor *crypto.Cryptor,
	tenantID uuid.UUID,
	resourceID uuid.UUID,
	alias string,
) (db.TenantResource, error) {
	row, err := q.GetResourceHeader(ctx, resourceID)
	if err != nil || row.TenantID != tenantID {
		if errors.Is(err, pgx.ErrNoRows) || err == nil {
			return db.TenantResource{}, ErrUnknownResource
		}
		return db.TenantResource{}, fmt.Errorf("get resource to duplicate: %w", err)
	}
	built, err := BuildResourceFieldsBatch(ctx, q, cryptor, []ResourceHeader{resourceHeaderFromGetRow(row)}, true)
	if err != nil {
		return db.TenantResource{}, fmt.Errorf("build resource snapshot: %w", err)
	}
	values := make(map[string]string, len(built[0].Fields))
	defer func() {
		for key := range values {
			delete(values, key)
		}
	}()
	for _, field := range built[0].Fields {
		if field.Value != "" {
			values[field.Key] = field.Value
		}
	}
	return provisionResource(ctx, q, cryptor, ProvisionResourceInput{
		Alias: alias, DefinitionKey: row.DefinitionKey, TenantID: tenantID, Values: values,
	})
}

func provisionResource(
	ctx context.Context,
	q ProvisionQuerier,
	cryptor *crypto.Cryptor,
	in ProvisionResourceInput,
) (db.TenantResource, error) {
	def, err := q.GetDefinitionByKey(ctx, in.DefinitionKey)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.TenantResource{}, ErrUnknownDefinition
		}
		return db.TenantResource{}, fmt.Errorf("get definition: %w", err)
	}
	if def.Status != "active" {
		return db.TenantResource{}, ErrInactiveDefinition
	}
	name, alias, err := NormalizeResourceIdentity(in.Name, in.Alias)
	if err != nil {
		return db.TenantResource{}, err
	}
	if alias == "" {
		alias = in.DefinitionKey
		if alias == "" {
			alias = def.Key
		}
	}
	if !resourceAliasPattern.MatchString(alias) {
		return db.TenantResource{}, ErrInvalidResourceAlias
	}
	if name == "" {
		name = alias
	}
	fields, err := q.ListFields(ctx, def.ID)
	if err != nil {
		return db.TenantResource{}, err
	}
	providedBySource := map[uuid.UUID]bool{}
	var sourceID pgtype.UUID
	if in.SourceResourceID != nil {
		source, err := q.GetResourceForLink(ctx, db.GetResourceForLinkParams{ID: *in.SourceResourceID, TenantID: in.TenantID})
		if err != nil || source.ResourceDefinitionID != def.ID || source.Status != "active" || source.SourceResourceID.Valid {
			return db.TenantResource{}, ErrInvalidResourceSource
		}
		values, err := q.ListResourceValues(ctx, source.ID)
		if err != nil {
			return db.TenantResource{}, fmt.Errorf("list source values: %w", err)
		}
		for _, value := range values {
			providedBySource[value.ResourceFieldID] = true
		}
		sourceID = pgtype.UUID{Bytes: source.ID, Valid: true}
	}
	for _, f := range fields {
		if value, ok := in.Values[f.Key]; ok && !ValidResourceValue(f.DataType, value) {
			return db.TenantResource{}, InvalidFieldValueError{Key: f.Key}
		}
		if f.Required {
			if v, ok := in.Values[f.Key]; (!ok || v == "") && !providedBySource[f.ID] {
				return db.TenantResource{}, MissingRequiredFieldError{Key: f.Key}
			}
		}
	}

	res, err := q.CreateTenantResource(ctx, db.CreateTenantResourceParams{
		TenantID: in.TenantID, ResourceDefinitionID: def.ID, Alias: alias, DisplayName: name, SourceResourceID: sourceID,
	})
	if err != nil {
		switch {
		case store.IsPostgresCode(err, store.PostgresUniqueViolation):
			return db.TenantResource{}, ErrResourceAliasExists
		case store.IsPostgresCode(err, store.PostgresForeignKeyViolation):
			return db.TenantResource{}, ErrUnknownTenant
		default:
			return db.TenantResource{}, fmt.Errorf("create tenant resource: %w", err)
		}
	}

	for _, f := range fields {
		raw, ok := in.Values[f.Key]
		if !ok {
			continue
		}
		p, err := encodeValue(cryptor, f.IsSecret, raw)
		if err != nil {
			return db.TenantResource{}, fmt.Errorf("encode field %q: %w", f.Key, err)
		}
		p.TenantResourceID = res.ID
		p.ResourceFieldID = f.ID
		if _, err := q.UpsertResourceValue(ctx, p); err != nil {
			return db.TenantResource{}, fmt.Errorf("store field %q: %w", f.Key, err)
		}
	}
	return res, nil
}

var resourceAliasPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,62}$`)

func NormalizeResourceIdentity(name, alias string) (string, string, error) {
	name = strings.TrimSpace(name)
	alias = strings.TrimSpace(alias)
	if alias != "" && !resourceAliasPattern.MatchString(alias) {
		return "", "", ErrInvalidResourceAlias
	}
	if len([]rune(name)) > 120 {
		return "", "", ErrInvalidResourceName
	}
	return name, alias, nil
}

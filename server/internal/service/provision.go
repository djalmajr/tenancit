package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/djalmajr/tenancit/server/internal/crypto"
	"github.com/djalmajr/tenancit/server/internal/store"
	"github.com/djalmajr/tenancit/server/internal/store/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var (
	ErrActiveResourceExists = errors.New("active resource of this type already exists")
	ErrInactiveDefinition   = errors.New("definition is inactive")
	ErrUnknownDefinition    = errors.New("unknown definition")
	ErrUnknownTenant        = errors.New("unknown tenant")
)

type MissingRequiredFieldError struct {
	Key string
}

func (e MissingRequiredFieldError) Error() string {
	return "missing required field: " + e.Key
}

type ProvisionResourceInput struct {
	DefinitionKey string
	TenantID      uuid.UUID
	Values        map[string]string
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
	UpsertResourceValue(ctx context.Context, arg db.UpsertResourceValueParams) (db.UpsertResourceValueRow, error)
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
	fields, err := q.ListFields(ctx, def.ID)
	if err != nil {
		return db.TenantResource{}, err
	}
	for _, f := range fields {
		if f.Required {
			if v, ok := in.Values[f.Key]; !ok || v == "" {
				return db.TenantResource{}, MissingRequiredFieldError{Key: f.Key}
			}
		}
	}

	res, err := q.CreateTenantResource(ctx, db.CreateTenantResourceParams{
		TenantID: in.TenantID, ResourceDefinitionID: def.ID,
	})
	if err != nil {
		switch {
		case store.IsPostgresCode(err, store.PostgresUniqueViolation):
			return db.TenantResource{}, ErrActiveResourceExists
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

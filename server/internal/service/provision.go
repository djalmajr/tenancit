package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/djalmajr/konvario/server/internal/crypto"
	"github.com/djalmajr/konvario/server/internal/store/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var (
	ErrActiveResourceExists = errors.New("active resource of this type already exists")
	ErrInactiveDefinition   = errors.New("definition is inactive")
	ErrUnknownDefinition    = errors.New("unknown definition")
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
	Cryptor   *crypto.Cryptor
	Queries   *db.Queries
	TxStarter ResourceTransactor
}

type ResourceTransactor interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

func ProvisionResource(ctx context.Context, deps ProvisionResourceDeps, in ProvisionResourceInput) (db.TenantResource, error) {
	tx, err := deps.TxStarter.Begin(ctx)
	if err != nil {
		return db.TenantResource{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	txDeps := ProvisionResourceDeps{Cryptor: deps.Cryptor, Queries: deps.Queries.WithTx(tx)}
	res, err := provisionResource(ctx, txDeps, in)
	if err != nil {
		return db.TenantResource{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return db.TenantResource{}, err
	}
	return res, nil
}

func provisionResource(
	ctx context.Context,
	deps ProvisionResourceDeps,
	in ProvisionResourceInput,
) (db.TenantResource, error) {
	def, err := deps.Queries.GetDefinitionByKey(ctx, in.DefinitionKey)
	if err != nil {
		return db.TenantResource{}, ErrUnknownDefinition
	}
	if def.Status != "active" {
		return db.TenantResource{}, ErrInactiveDefinition
	}
	fields, err := deps.Queries.ListFields(ctx, def.ID)
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

	res, err := deps.Queries.CreateTenantResource(ctx, db.CreateTenantResourceParams{
		TenantID: in.TenantID, ResourceDefinitionID: def.ID,
	})
	if err != nil {
		return db.TenantResource{}, ErrActiveResourceExists
	}

	for _, f := range fields {
		raw, ok := in.Values[f.Key]
		if !ok {
			continue
		}
		p, err := encodeValue(deps.Cryptor, f.IsSecret, raw)
		if err != nil {
			return db.TenantResource{}, fmt.Errorf("encode field %q: %w", f.Key, err)
		}
		p.TenantResourceID = res.ID
		p.ResourceFieldID = f.ID
		if _, err := deps.Queries.UpsertResourceValue(ctx, p); err != nil {
			return db.TenantResource{}, fmt.Errorf("store field %q: %w", f.Key, err)
		}
	}
	return res, nil
}

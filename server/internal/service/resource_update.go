package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/djalmajr/tenancit/server/internal/crypto"
	"github.com/djalmajr/tenancit/server/internal/store/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var (
	ErrUnknownResource      = errors.New("unknown resource")
	ErrUnknownResourceField = errors.New("unknown resource field")
)

type UpdateResourceFieldInput struct {
	FieldKey   string
	ResourceID uuid.UUID
	TenantID   uuid.UUID
	Value      string
}

type UpdateResourceFieldQuerier interface {
	GetTenantResource(ctx context.Context, arg db.GetTenantResourceParams) (db.TenantResource, error)
	ListFields(ctx context.Context, resourceDefinitionID uuid.UUID) ([]db.ResourceField, error)
	UpsertResourceValue(ctx context.Context, arg db.UpsertResourceValueParams) (db.UpsertResourceValueRow, error)
}

func UpdateResourceFieldInTx(
	ctx context.Context,
	q UpdateResourceFieldQuerier,
	cryptor *crypto.Cryptor,
	in UpdateResourceFieldInput,
) (db.ResourceField, error) {
	resource, err := q.GetTenantResource(ctx, db.GetTenantResourceParams{ID: in.ResourceID, TenantID: in.TenantID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.ResourceField{}, ErrUnknownResource
		}
		return db.ResourceField{}, err
	}
	fields, err := q.ListFields(ctx, resource.ResourceDefinitionID)
	if err != nil {
		return db.ResourceField{}, err
	}
	var selected *db.ResourceField
	for i := range fields {
		if fields[i].Key == in.FieldKey {
			selected = &fields[i]
			break
		}
	}
	if selected == nil {
		return db.ResourceField{}, ErrUnknownResourceField
	}
	if selected.Required && in.Value == "" {
		return db.ResourceField{}, MissingRequiredFieldError{Key: selected.Key}
	}
	params, err := encodeValue(cryptor, selected.IsSecret, in.Value)
	if err != nil {
		return db.ResourceField{}, fmt.Errorf("encode field %q: %w", selected.Key, err)
	}
	params.TenantResourceID = resource.ID
	params.ResourceFieldID = selected.ID
	if _, err := q.UpsertResourceValue(ctx, params); err != nil {
		return db.ResourceField{}, fmt.Errorf("store field %q: %w", selected.Key, err)
	}
	return *selected, nil
}

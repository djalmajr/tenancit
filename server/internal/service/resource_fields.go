package service

import (
	"context"

	"github.com/djalmajr/konvario/server/internal/crypto"
	"github.com/djalmajr/konvario/server/internal/store/db"
	"github.com/google/uuid"
)

type ResourceFieldValue struct {
	DataType string `json:"dataType"`
	IsSecret bool   `json:"isSecret"`
	Key      string `json:"key"`
	Label    string `json:"label"`
	Required bool   `json:"required"`
	Value    string `json:"value"`
}

type ResourceFieldQuerier interface {
	GetDefinition(ctx context.Context, id uuid.UUID) (db.ResourceDefinition, error)
	ListFields(ctx context.Context, resourceDefinitionID uuid.UUID) ([]db.ResourceField, error)
	ListResourceValues(ctx context.Context, tenantResourceID uuid.UUID) ([]db.TenantResourceValue, error)
}

type BuildResourceFieldsDeps struct {
	Cryptor *crypto.Cryptor
	Queries ResourceFieldQuerier
}

type BuildResourceFieldsInput struct {
	Resource db.TenantResource
	Reveal   bool
}

type BuildResourceFieldsResult struct {
	Definition db.ResourceDefinition
	Fields     []ResourceFieldValue
}

func BuildResourceFields(ctx context.Context, deps BuildResourceFieldsDeps, in BuildResourceFieldsInput) (BuildResourceFieldsResult, error) {
	def, err := deps.Queries.GetDefinition(ctx, in.Resource.ResourceDefinitionID)
	if err != nil {
		return BuildResourceFieldsResult{}, err
	}
	fields, err := deps.Queries.ListFields(ctx, def.ID)
	if err != nil {
		return BuildResourceFieldsResult{}, err
	}
	values, err := deps.Queries.ListResourceValues(ctx, in.Resource.ID)
	if err != nil {
		return BuildResourceFieldsResult{}, err
	}
	valueByField := make(map[uuid.UUID]db.TenantResourceValue, len(values))
	for _, v := range values {
		valueByField[v.ResourceFieldID] = v
	}

	out := BuildResourceFieldsResult{Definition: def, Fields: make([]ResourceFieldValue, 0, len(fields))}
	for _, f := range fields {
		fv := ResourceFieldValue{
			DataType: f.DataType,
			IsSecret: f.IsSecret,
			Key:      f.Key,
			Label:    f.Label,
			Required: f.Required,
		}
		if v, ok := valueByField[f.ID]; ok {
			shown, err := presentValue(deps.Cryptor, f.IsSecret, v, in.Reveal)
			if err != nil {
				return BuildResourceFieldsResult{}, err
			}
			fv.Value = shown
		}
		out.Fields = append(out.Fields, fv)
	}
	return out, nil
}

package service

import (
	"context"
	"time"

	"github.com/djalmajr/tenancit/server/internal/crypto"
	"github.com/djalmajr/tenancit/server/internal/store/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type ResourceFieldValue struct {
	DataType   string `json:"dataType"`
	IsSecret   bool   `json:"isSecret"`
	Key        string `json:"key"`
	Label      string `json:"label"`
	Required   bool   `json:"required"`
	IsOverride bool   `json:"isOverride"`
	Origin     string `json:"origin"`
	Value      string `json:"value"`
}

type ResourceHeader struct {
	Resource            db.TenantResource
	DefinitionKey       string
	DefinitionName      string
	DefinitionUpdatedAt time.Time
	SourceAlias         string
	SourceUpdatedAt     time.Time
}

type BuiltResourceFields struct {
	Header ResourceHeader
	Fields []ResourceFieldValue
}

type ResourceBatchQuerier interface {
	ListResourceFieldValuesByResourceIDs(ctx context.Context, resourceIds []uuid.UUID) ([]db.ListResourceFieldValuesByResourceIDsRow, error)
}

type ResourceHeaderQuerier interface {
	ListResourceHeadersByTenant(ctx context.Context, arg db.ListResourceHeadersByTenantParams) ([]db.ListResourceHeadersByTenantRow, error)
}

func LoadResourceHeaders(
	ctx context.Context,
	q ResourceHeaderQuerier,
	tenantID uuid.UUID,
	includeInactive bool,
) ([]ResourceHeader, error) {
	rows, err := q.ListResourceHeadersByTenant(ctx, db.ListResourceHeadersByTenantParams{
		TenantID: tenantID, IncludeInactive: includeInactive,
	})
	if err != nil {
		return nil, err
	}
	headers := make([]ResourceHeader, 0, len(rows))
	for _, row := range rows {
		headers = append(headers, resourceHeaderFromListRow(row))
	}
	return headers, nil
}

func BuildResourceFieldsBatch(
	ctx context.Context,
	q ResourceBatchQuerier,
	cryptor *crypto.Cryptor,
	headers []ResourceHeader,
	reveal bool,
) ([]BuiltResourceFields, error) {
	out := make([]BuiltResourceFields, len(headers))
	ids := make([]uuid.UUID, len(headers))
	indexByID := make(map[uuid.UUID]int, len(headers))
	for i, header := range headers {
		out[i] = BuiltResourceFields{Header: header, Fields: []ResourceFieldValue{}}
		ids[i] = header.Resource.ID
		indexByID[header.Resource.ID] = i
	}
	if len(ids) == 0 {
		return out, nil
	}

	rows, err := q.ListResourceFieldValuesByResourceIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		index, ok := indexByID[row.TenantResourceID]
		if !ok {
			continue
		}
		fv := ResourceFieldValue{
			DataType:   row.DataType,
			IsSecret:   row.IsSecret,
			Key:        row.FieldKey,
			Label:      row.FieldLabel,
			Required:   row.Required,
			IsOverride: row.IsOverride,
		}
		if row.IsOverride || !out[index].Header.Resource.SourceResourceID.Valid {
			fv.Origin = "local"
		} else {
			fv.Origin = "inherited"
		}
		if row.HasValue {
			value := db.TenantResourceValue{
				TenantResourceID: row.TenantResourceID,
				ResourceFieldID:  row.ResourceFieldID,
				ValuePlain:       row.ValuePlain,
				ValueCipher:      row.ValueCipher,
				Nonce:            row.Nonce,
				KeyVersion:       row.KeyVersion,
			}
			shown, err := presentValue(cryptor, row.IsSecret, value, reveal)
			if err != nil {
				return nil, err
			}
			fv.Value = shown
		}
		out[index].Fields = append(out[index].Fields, fv)
	}
	return out, nil
}

func resourceHeaderFromListRow(row db.ListResourceHeadersByTenantRow) ResourceHeader {
	return ResourceHeader{
		Resource: db.TenantResource{
			ID: row.ID, TenantID: row.TenantID, ResourceDefinitionID: row.ResourceDefinitionID,
			Status: row.Status, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
			Alias: row.Alias, SourceResourceID: row.SourceResourceID,
		},
		DefinitionKey:       row.DefinitionKey,
		DefinitionName:      row.DefinitionName,
		DefinitionUpdatedAt: row.DefinitionUpdatedAt,
		SourceAlias:         stringValue(row.SourceAlias),
		SourceUpdatedAt:     timeValue(row.SourceUpdatedAt),
	}
}

func resourceHeaderFromGetRow(row db.GetResourceHeaderRow) ResourceHeader {
	return ResourceHeader{
		Resource: db.TenantResource{
			ID: row.ID, TenantID: row.TenantID, ResourceDefinitionID: row.ResourceDefinitionID,
			Status: row.Status, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
			Alias: row.Alias, SourceResourceID: row.SourceResourceID,
		},
		DefinitionKey:       row.DefinitionKey,
		DefinitionName:      row.DefinitionName,
		DefinitionUpdatedAt: row.DefinitionUpdatedAt,
		SourceAlias:         stringValue(row.SourceAlias),
		SourceUpdatedAt:     timeValue(row.SourceUpdatedAt),
	}
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func timeValue(value pgtype.Timestamptz) time.Time {
	if !value.Valid {
		return time.Time{}
	}
	return value.Time
}

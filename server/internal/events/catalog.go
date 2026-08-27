package events

import (
	"encoding/json"
	"fmt"
)

const SchemaVersion = 1

type Draft struct {
	Type          string
	Version       int32
	AggregateType string
	AggregateID   string
	Payload       []byte
}

var publishedActions = map[string]string{
	"tenant.created":            "tenant.created",
	"tenant.updated":            "tenant.updated",
	"tenant.deleted":            "tenant.deleted",
	"domain.added":              "tenant_domain.added",
	"domain.updated":            "tenant_domain.updated",
	"domain.deleted":            "tenant_domain.deleted",
	"definition.created":        "resource_definition.created",
	"definition.updated":        "resource_definition.updated",
	"definition.deleted":        "resource_definition.deleted",
	"definition.status_changed": "resource_definition.status_changed",
	"definition.field_added":    "resource_field.added",
	"definition.field_updated":  "resource_field.updated",
	"definition.field_deleted":  "resource_field.deleted",
	"resource.provisioned":      "tenant_resource.provisioned",
	"resource.updated":          "tenant_resource.updated",
	"resource.status_changed":   "tenant_resource.status_changed",
	"resource.deleted":          "tenant_resource.deleted",
	"api_client.created":        "api_client.created",
	"api_client.policy_updated": "api_client.policy_updated",
	"api_client.rotated":        "api_client.rotated",
	"api_client.revoked":        "api_client.revoked",
	"api_client.deleted":        "api_client.deleted",
}

func FromAudit(action, aggregateType, aggregateID string) (Draft, bool, error) {
	eventName, ok := publishedActions[action]
	if !ok {
		return Draft{}, false, nil
	}
	payload, err := json.Marshal(map[string]any{
		"schema_version": SchemaVersion,
		"resource":       map[string]string{"type": aggregateType, "id": aggregateID},
	})
	if err != nil {
		return Draft{}, false, fmt.Errorf("marshal event reference: %w", err)
	}
	return Draft{
		Type: "tenancit." + eventName, Version: SchemaVersion,
		AggregateType: aggregateType, AggregateID: aggregateID, Payload: payload,
	}, true, nil
}

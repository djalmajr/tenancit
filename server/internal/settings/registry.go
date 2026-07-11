package settings

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

type ValueType string

const (
	TypeInteger ValueType = "integer"
	TypeEnum    ValueType = "enum"
)

const (
	SessionAbsoluteHours         = "session_absolute_hours"
	SessionIdleMinutes           = "session_idle_minutes"
	APIClientDefaultRPM          = "api_client_default_rpm"
	APIClientDefaultTTLDays      = "api_client_default_ttl_days"
	UsageRetentionMonths         = "usage_retention_months"
	AuditRetentionDays           = "audit_retention_days"
	DefaultLocale                = "default_locale"
	WebhookDeliveryRetentionDays = "webhook_delivery_retention_days"
	OutboxEventRetentionDays     = "outbox_event_retention_days"
)

type Definition struct {
	Key          string    `json:"key"`
	Type         ValueType `json:"type"`
	DefaultValue string    `json:"default_value"`
	Owner        string    `json:"owner"`
	Minimum      *int      `json:"minimum,omitempty"`
	Maximum      *int      `json:"maximum,omitempty"`
	Options      []string  `json:"options,omitempty"`
}

func integerDefinition(key, defaultValue, owner string, minimum, maximum int) Definition {
	return Definition{Key: key, Type: TypeInteger, DefaultValue: defaultValue, Owner: owner, Minimum: &minimum, Maximum: &maximum}
}

var registry = map[string]Definition{
	SessionAbsoluteHours:         integerDefinition(SessionAbsoluteHours, "8", "security", 1, 24),
	SessionIdleMinutes:           integerDefinition(SessionIdleMinutes, "30", "security", 5, 120),
	APIClientDefaultRPM:          integerDefinition(APIClientDefaultRPM, "300", "platform", 1, 10000),
	APIClientDefaultTTLDays:      integerDefinition(APIClientDefaultTTLDays, "90", "platform", 1, 365),
	UsageRetentionMonths:         integerDefinition(UsageRetentionMonths, "6", "operations", 1, 24),
	WebhookDeliveryRetentionDays: integerDefinition(WebhookDeliveryRetentionDays, "30", "operations", 7, 365),
	OutboxEventRetentionDays:     integerDefinition(OutboxEventRetentionDays, "90", "operations", 30, 730),
	AuditRetentionDays:           integerDefinition(AuditRetentionDays, "180", "security", 30, 730),
	DefaultLocale: {
		Key: DefaultLocale, Type: TypeEnum, DefaultValue: "pt-BR", Owner: "product",
		Options: []string{"pt-BR", "en-US", "es-ES"},
	},
}

func Definitions() []Definition {
	definitions := make([]Definition, 0, len(registry))
	for _, definition := range registry {
		copy := definition
		copy.Options = append([]string(nil), definition.Options...)
		definitions = append(definitions, copy)
	}
	sort.Slice(definitions, func(i, j int) bool { return definitions[i].Key < definitions[j].Key })
	return definitions
}

func Defaults() map[string]string {
	values := make(map[string]string, len(registry))
	for key, definition := range registry {
		values[key] = definition.DefaultValue
	}
	return values
}

func Validate(updates map[string]string, current map[string]string) (map[string]string, error) {
	if len(updates) == 0 {
		return nil, errors.New("at least one setting is required")
	}
	normalized := make(map[string]string, len(updates))
	for key, raw := range updates {
		definition, ok := registry[key]
		if !ok {
			return nil, fmt.Errorf("unknown setting %q", key)
		}
		value := strings.TrimSpace(raw)
		switch definition.Type {
		case TypeInteger:
			number, err := strconv.Atoi(value)
			if err != nil || number < *definition.Minimum || number > *definition.Maximum {
				return nil, fmt.Errorf("setting %q is outside its allowed range", key)
			}
			value = strconv.Itoa(number)
		case TypeEnum:
			if !contains(definition.Options, value) {
				return nil, fmt.Errorf("setting %q contains an unsupported value", key)
			}
		default:
			return nil, fmt.Errorf("setting %q has an unsupported type", key)
		}
		normalized[key] = value
	}
	merged := Defaults()
	for key, value := range current {
		if _, known := registry[key]; known {
			merged[key] = value
		}
	}
	for key, value := range normalized {
		merged[key] = value
	}
	absoluteHours, _ := strconv.Atoi(merged[SessionAbsoluteHours])
	idleMinutes, _ := strconv.Atoi(merged[SessionIdleMinutes])
	if idleMinutes > absoluteHours*60 {
		return nil, errors.New("session idle duration cannot exceed absolute duration")
	}
	deliveryDays, _ := strconv.Atoi(merged[WebhookDeliveryRetentionDays])
	eventDays, _ := strconv.Atoi(merged[OutboxEventRetentionDays])
	if eventDays < deliveryDays {
		return nil, errors.New("outbox event retention cannot be shorter than delivery retention")
	}
	return normalized, nil
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

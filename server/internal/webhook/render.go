package webhook

import (
	"encoding/json"
	"errors"
	"time"
)

type Event struct {
	ID, Type, AggregateType, AggregateID, RequestID string
	Version                                         int32
	OccurredAt                                      time.Time
	Payload                                         json.RawMessage
}

type envelope struct {
	ID            string          `json:"id"`
	Type          string          `json:"type"`
	Version       int32           `json:"version"`
	OccurredAt    time.Time       `json:"occurred_at"`
	AggregateType string          `json:"aggregate_type"`
	AggregateID   string          `json:"aggregate_id"`
	RequestID     string          `json:"request_id"`
	Data          json.RawMessage `json:"data"`
}

func Render(format string, event Event) ([]byte, error) {
	base := envelope{
		ID: event.ID, Type: event.Type, Version: event.Version, OccurredAt: event.OccurredAt.UTC(),
		AggregateType: event.AggregateType, AggregateID: event.AggregateID,
		RequestID: event.RequestID, Data: event.Payload,
	}
	switch format {
	case "generic":
		return json.Marshal(base)
	case "slack", "discord":
		return json.Marshal(map[string]any{"text": event.Type, "event": base})
	case "teams":
		return json.Marshal(map[string]any{
			"type": "message", "attachments": []any{map[string]any{
				"contentType": "application/vnd.microsoft.card.adaptive",
				"content":     map[string]any{"type": "AdaptiveCard", "version": "1.4", "body": []any{map[string]string{"type": "TextBlock", "text": event.Type}}, "event": base},
			}},
		})
	default:
		return nil, errors.New("unsupported webhook format")
	}
}

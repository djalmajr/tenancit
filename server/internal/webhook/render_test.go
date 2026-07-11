package webhook

import (
	"strings"
	"testing"
	"time"
)

func TestRenderersPreserveVersionedReferenceEnvelope(t *testing.T) {
	event := Event{ID: "event-1", Type: "tenancit.tenant.created", Version: 1, AggregateType: "tenant", AggregateID: "tenant-1", RequestID: "request-1", OccurredAt: time.Unix(1, 0).UTC(), Payload: []byte(`{"schema_version":1,"resource":{"type":"tenant","id":"tenant-1"}}`)}
	for _, format := range []string{"generic", "slack", "discord", "teams"} {
		body, err := Render(format, event)
		if err != nil {
			t.Fatalf("render %s: %v", format, err)
		}
		if !strings.Contains(string(body), "tenancit.tenant.created") || strings.Contains(strings.ToLower(string(body)), "password") {
			t.Fatalf("unsafe %s body=%s", format, body)
		}
	}
}

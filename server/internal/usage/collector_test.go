package usage

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestAggregateEventsGroupsByUTCDateOperationAndStatusClass(t *testing.T) {
	clientID := uuid.New()
	at := time.Date(2026, 7, 10, 23, 59, 0, 0, time.FixedZone("offset", -3*60*60))
	events := []Event{
		{APIClientID: clientID, Operation: "resolve", Status: 200, At: at},
		{APIClientID: clientID, Operation: "resolve", Status: 204, At: at.Add(time.Minute)},
		{APIClientID: clientID, Operation: "resolve", Status: 404, At: at},
	}

	batch := aggregateEvents(events)
	if len(batch.Usage) != 2 {
		t.Fatalf("usage groups = %d, want 2", len(batch.Usage))
	}
	if got := batch.LastUsed[clientID]; !got.Equal(at.Add(time.Minute)) {
		t.Fatalf("last used = %v, want %v", got, at.Add(time.Minute))
	}
	for key, count := range batch.Usage {
		if key.Day != "2026-07-11" || count.Requests == 0 {
			t.Fatalf("unexpected aggregate: key=%+v count=%+v", key, count)
		}
	}
}

func TestAggregateEventsSeparatesRateLimitedCount(t *testing.T) {
	clientID := uuid.New()
	batch := aggregateEvents([]Event{{
		APIClientID: clientID, Operation: "identify", Status: 429,
		RateLimited: true, At: time.Date(2026, 7, 10, 1, 0, 0, 0, time.UTC),
	}})
	for _, count := range batch.Usage {
		if count.Requests != 0 || count.RateLimited != 1 {
			t.Fatalf("count = %+v, want only one rate-limited request", count)
		}
	}
}

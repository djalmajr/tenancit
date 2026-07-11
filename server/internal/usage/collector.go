package usage

import (
	"context"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/djalmajr/tenancit/server/internal/store/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type Event struct {
	APIClientID uuid.UUID
	Operation   string
	Status      int
	RateLimited bool
	At          time.Time
}

type usageKey struct {
	Day         string
	APIClientID uuid.UUID
	Operation   string
	StatusClass int16
}

type usageCount struct {
	Requests    int64
	RateLimited int64
}

type eventBatch struct {
	LastUsed map[uuid.UUID]time.Time
	Usage    map[usageKey]usageCount
}

type Store interface {
	TouchAPIClientLastUsed(context.Context, db.TouchAPIClientLastUsedParams) error
	UpsertAPIClientUsageDaily(context.Context, db.UpsertAPIClientUsageDailyParams) error
}

type Collector struct {
	store         Store
	events        chan Event
	flushInterval time.Duration
	dropped       atomic.Uint64
}

func NewCollector(store Store, capacity int, flushInterval time.Duration) *Collector {
	return &Collector{store: store, events: make(chan Event, capacity), flushInterval: flushInterval}
}

func (c *Collector) Record(event Event) {
	if event.At.IsZero() {
		event.At = time.Now().UTC()
	}
	select {
	case c.events <- event:
	default:
		c.dropped.Add(1)
	}
}

func (c *Collector) Dropped() uint64 { return c.dropped.Load() }

func (c *Collector) Run(ctx context.Context) {
	ticker := time.NewTicker(c.flushInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			c.flush(context.Background())
			return
		case <-ticker.C:
			c.flush(ctx)
		}
	}
}

func (c *Collector) flush(ctx context.Context) {
	events := make([]Event, 0, len(c.events))
	for {
		select {
		case event := <-c.events:
			events = append(events, event)
		default:
			goto drained
		}
	}

drained:
	batch := aggregateEvents(events)
	for clientID, usedAt := range batch.LastUsed {
		if err := c.store.TouchAPIClientLastUsed(ctx, db.TouchAPIClientLastUsedParams{
			ApiClientID: clientID,
			UsedAt:      usedAt.UTC(),
		}); err != nil {
			slog.Error("flush API client last used", "api_client_id", clientID, "err", err)
		}
	}
	for key, count := range batch.Usage {
		day, err := time.Parse("2006-01-02", key.Day)
		if err != nil {
			continue
		}
		if err := c.store.UpsertAPIClientUsageDaily(ctx, db.UpsertAPIClientUsageDailyParams{
			Day: pgtype.Date{Time: day, Valid: true}, ApiClientID: key.APIClientID,
			Operation: key.Operation, StatusClass: key.StatusClass,
			RequestCount: count.Requests, RateLimitedCount: count.RateLimited,
		}); err != nil {
			slog.Error("flush API client usage", "api_client_id", key.APIClientID, "err", err)
		}
	}
}

func aggregateEvents(events []Event) eventBatch {
	batch := eventBatch{
		LastUsed: make(map[uuid.UUID]time.Time),
		Usage:    make(map[usageKey]usageCount),
	}
	for _, event := range events {
		if current := batch.LastUsed[event.APIClientID]; event.At.After(current) {
			batch.LastUsed[event.APIClientID] = event.At
		}
		statusClass := int16(event.Status / 100)
		key := usageKey{
			Day: event.At.UTC().Format("2006-01-02"), APIClientID: event.APIClientID,
			Operation: event.Operation, StatusClass: statusClass,
		}
		count := batch.Usage[key]
		if event.RateLimited {
			count.RateLimited++
		} else {
			count.Requests++
		}
		batch.Usage[key] = count
	}
	return batch
}

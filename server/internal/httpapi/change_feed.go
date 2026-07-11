package httpapi

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

type changeFeedEvent struct {
	ID            string          `json:"id"`
	Type          string          `json:"type"`
	Version       int32           `json:"version"`
	AggregateType string          `json:"aggregate_type"`
	AggregateID   string          `json:"aggregate_id"`
	OccurredAt    time.Time       `json:"occurred_at"`
	Data          json.RawMessage `json:"data"`
}

type changeFeedResponse struct {
	Events     []changeFeedEvent `json:"events"`
	NextCursor string            `json:"next_cursor,omitempty"`
}

func encodeFeedCursor(occurredAt time.Time, id string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(occurredAt.UTC().Format(time.RFC3339Nano) + "|" + id))
}

func decodeFeedCursor(value string) (time.Time, uuid.UUID, error) {
	if value == "" {
		return time.Time{}, uuid.Nil, nil
	}
	decoded, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return time.Time{}, uuid.Nil, errors.New("invalid cursor")
	}
	parts := strings.SplitN(string(decoded), "|", 2)
	if len(parts) != 2 {
		return time.Time{}, uuid.Nil, errors.New("invalid cursor")
	}
	when, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return time.Time{}, uuid.Nil, errors.New("invalid cursor")
	}
	id, err := uuid.Parse(parts[1])
	if err != nil {
		return time.Time{}, uuid.Nil, errors.New("invalid cursor")
	}
	return when.UTC(), id, nil
}

func (s *Server) listChangeFeed(w http.ResponseWriter, r *http.Request) {
	limit := 100
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 200 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_limit"})
			return
		}
		limit = parsed
	}
	cursorTime, cursorID, err := decodeFeedCursor(r.URL.Query().Get("cursor"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_cursor"})
		return
	}
	rows, err := s.DB.Query(r.Context(), `SELECT id,event_type,event_version,aggregate_type,aggregate_id,occurred_at,payload
		FROM outbox_events WHERE ($1::boolean=false OR (occurred_at,id)>($2,$3)) ORDER BY occurred_at,id LIMIT $4`, !cursorTime.IsZero(), cursorTime, cursorID, limit)
	if err != nil {
		writeInternalError(w, r, "list change feed", err)
		return
	}
	defer rows.Close()
	events := []changeFeedEvent{}
	for rows.Next() {
		var event changeFeedEvent
		if err := rows.Scan(&event.ID, &event.Type, &event.Version, &event.AggregateType, &event.AggregateID, &event.OccurredAt, &event.Data); err != nil {
			writeInternalError(w, r, "scan change feed", err)
			return
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		writeInternalError(w, r, "read change feed", err)
		return
	}
	response := changeFeedResponse{Events: events}
	if len(events) == limit {
		last := events[len(events)-1]
		response.NextCursor = encodeFeedCursor(last.OccurredAt, last.ID)
	}
	w.Header().Set("Cache-Control", "private, no-store")
	writeJSON(w, http.StatusOK, response)
}

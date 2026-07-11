package httpapi

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/djalmajr/tenancit/server/internal/store/db"
	"github.com/google/uuid"
)

const (
	defaultAuditPageLimit = 50
	maxAuditPageLimit     = 200
	maxAuditWindow        = 31 * 24 * time.Hour
)

type auditCursor struct {
	OccurredAt time.Time `json:"occurred_at"`
	ID         uuid.UUID `json:"id"`
}

type auditEventPage struct {
	Events     []auditEventView `json:"events"`
	NextCursor string           `json:"next_cursor,omitempty"`
}

type auditEventView struct {
	db.AdminAuditEvent
	Metadata json.RawMessage `json:"metadata"`
}

type auditReadMetadata struct {
	From      time.Time `json:"from"`
	To        time.Time `json:"to"`
	Rows      int       `json:"rows"`
	HasCursor bool      `json:"has_cursor"`
}

func (s *Server) listAuditEvents(w http.ResponseWriter, r *http.Request) {
	now := time.Now().UTC()
	from, err := auditTimeParam(r, "from", now.Add(-24*time.Hour))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid from"})
		return
	}
	to, err := auditTimeParam(r, "to", now)
	if err != nil || !to.After(from) || to.Sub(from) > maxAuditWindow {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid audit window"})
		return
	}
	limit, err := auditLimit(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid limit"})
		return
	}
	cursor, hasCursor, err := decodeAuditCursor(r.URL.Query().Get("cursor"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid cursor"})
		return
	}

	tx, err := s.DB.Begin(r.Context())
	if err != nil {
		writeInternalError(w, r, "begin audit read", err)
		return
	}
	defer tx.Rollback(r.Context())
	q := db.New(tx)
	events, err := q.ListAdminAuditEvents(r.Context(), db.ListAdminAuditEventsParams{
		FromTime: from, ToTime: to,
		ActorKind: r.URL.Query().Get("actor_kind"), ActorSubject: r.URL.Query().Get("actor_subject"),
		Action: r.URL.Query().Get("action"), TargetType: r.URL.Query().Get("target_type"),
		TargetID: r.URL.Query().Get("target_id"), RequestID: r.URL.Query().Get("request_id"),
		Result: r.URL.Query().Get("result"), HasCursor: hasCursor,
		CursorTime: cursor.OccurredAt, CursorID: cursor.ID, PageLimit: int32(limit),
	})
	if err != nil {
		writeInternalError(w, r, "list admin audit events", err)
		return
	}
	if err := insertAdminAuditSuccess(r, q, "audit.events_read", "audit_query", "activity", "/v1/admin/audit-events", http.StatusOK, auditReadMetadata{From: from, To: to, Rows: len(events), HasCursor: hasCursor}); err != nil {
		writeInternalError(w, r, "audit audit read", err)
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeInternalError(w, r, "commit audit read", err)
		return
	}
	views := make([]auditEventView, 0, len(events))
	for _, event := range events {
		views = append(views, auditEventView{AdminAuditEvent: event, Metadata: json.RawMessage(event.Metadata)})
	}
	page := auditEventPage{Events: views}
	if len(events) == limit {
		last := events[len(events)-1]
		page.NextCursor = encodeAuditCursor(auditCursor{OccurredAt: last.OccurredAt, ID: last.ID})
	}
	writeJSON(w, http.StatusOK, page)
}

func auditTimeParam(r *http.Request, name string, fallback time.Time) (time.Time, error) {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		return fallback, nil
	}
	return time.Parse(time.RFC3339, raw)
}

func auditLimit(r *http.Request) (int, error) {
	raw := r.URL.Query().Get("limit")
	if raw == "" {
		return defaultAuditPageLimit, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 1 || value > maxAuditPageLimit {
		return 0, strconv.ErrRange
	}
	return value, nil
}

func encodeAuditCursor(cursor auditCursor) string {
	data, _ := json.Marshal(cursor)
	return base64.RawURLEncoding.EncodeToString(data)
}

func decodeAuditCursor(raw string) (auditCursor, bool, error) {
	if raw == "" {
		return auditCursor{}, false, nil
	}
	data, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return auditCursor{}, false, err
	}
	var cursor auditCursor
	if err := json.Unmarshal(data, &cursor); err != nil || cursor.ID == uuid.Nil || cursor.OccurredAt.IsZero() {
		if err == nil {
			err = strconv.ErrSyntax
		}
		return auditCursor{}, false, err
	}
	return cursor, true, nil
}

package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/djalmajr/tenancit/server/internal/store/db"
	"github.com/djalmajr/tenancit/server/internal/webhook"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type webhookDeliveryView struct {
	ID             string     `json:"id"`
	EventID        string     `json:"event_id"`
	EventType      string     `json:"event_type"`
	TargetID       string     `json:"target_id"`
	TargetName     string     `json:"target_name"`
	Status         string     `json:"status"`
	AttemptCount   int32      `json:"attempt_count"`
	NextAttemptAt  time.Time  `json:"next_attempt_at"`
	LastHTTPStatus *int32     `json:"last_http_status,omitempty"`
	LastErrorCode  *string    `json:"last_error_code,omitempty"`
	DeliveredAt    *time.Time `json:"delivered_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}

type webhookOverview struct {
	Targets      int64 `json:"targets"`
	Pending      int64 `json:"pending"`
	Retry        int64 `json:"retry"`
	Delivered    int64 `json:"delivered"`
	DeadLetter   int64 `json:"dead_letter"`
	OpenCircuits int64 `json:"open_circuits"`
}

func (s *Server) webhookOverview(w http.ResponseWriter, r *http.Request) {
	var result webhookOverview
	err := s.DB.QueryRow(r.Context(), `SELECT
		(SELECT count(*) FROM webhook_targets WHERE status='active'),
		count(*) FILTER(WHERE status IN ('pending','delivering')),
		count(*) FILTER(WHERE status='retry'),count(*) FILTER(WHERE status='delivered'),
		count(*) FILTER(WHERE status='dead_letter'),
		(SELECT count(*) FROM webhook_targets WHERE circuit_open_until>clock_timestamp())
		FROM webhook_deliveries`).Scan(&result.Targets, &result.Pending, &result.Retry, &result.Delivered, &result.DeadLetter, &result.OpenCircuits)
	if err != nil {
		writeInternalError(w, r, "webhook overview", err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

type webhookTargetBody struct {
	Name   string `json:"name"`
	URL    string `json:"url"`
	Format string `json:"format"`
}
type webhookTargetStatusBody struct {
	Status string `json:"status"`
}
type webhookTargetAuditMetadata struct {
	Format string `json:"format"`
	Status string `json:"status"`
}

func (s *Server) listWebhookTargets(w http.ResponseWriter, r *http.Request) {
	if s.Webhooks == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "webhook_governance_unavailable"})
		return
	}
	targets, err := s.Webhooks.List(r.Context())
	if err != nil {
		writeInternalError(w, r, "list webhook targets", err)
		return
	}
	writeJSON(w, http.StatusOK, targets)
}

func (s *Server) createWebhookTarget(w http.ResponseWriter, r *http.Request) {
	var body webhookTargetBody
	if !decodeJSON(w, r, &body) {
		return
	}
	if s.Webhooks == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "webhook_governance_unavailable"})
		return
	}
	tx, err := s.DB.Begin(r.Context())
	if err != nil {
		writeInternalError(w, r, "begin webhook target create", err)
		return
	}
	defer tx.Rollback(r.Context())
	created, err := s.Webhooks.CreateTx(r.Context(), tx, body.Name, body.URL, body.Format)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_webhook_target"})
		return
	}
	if err := insertAdminAuditSuccess(r, db.New(tx), "webhook_target.created", "webhook_target", created.ID.String(), "/v1/admin/webhook-targets", http.StatusCreated, webhookTargetAuditMetadata{Format: created.Format, Status: created.Status}); err != nil {
		writeInternalError(w, r, "audit webhook target create", err)
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeInternalError(w, r, "commit webhook target create", err)
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	writeJSON(w, http.StatusCreated, created)
}

func (s *Server) setWebhookTargetStatus(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_webhook_target"})
		return
	}
	var body webhookTargetStatusBody
	if !decodeJSON(w, r, &body) {
		return
	}
	body.Status = strings.TrimSpace(body.Status)
	if s.Webhooks == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "webhook_governance_unavailable"})
		return
	}
	tx, err := s.DB.Begin(r.Context())
	if err != nil {
		writeInternalError(w, r, "begin webhook target status", err)
		return
	}
	defer tx.Rollback(r.Context())
	if err := s.Webhooks.SetStatusTx(r.Context(), tx, id, body.Status); errors.Is(err, webhook.ErrTargetNotFound) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "webhook_target_not_found"})
		return
	} else if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_webhook_target_status"})
		return
	}
	if err := insertAdminAuditSuccess(r, db.New(tx), "webhook_target.status_changed", "webhook_target", id.String(), "/v1/admin/webhook-targets/{id}/status", http.StatusNoContent, webhookTargetAuditMetadata{Status: body.Status}); err != nil {
		writeInternalError(w, r, "audit webhook target status", err)
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeInternalError(w, r, "commit webhook target status", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) listWebhookDeliveries(w http.ResponseWriter, r *http.Request) {
	limit := 100
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 200 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_limit"})
			return
		}
		limit = parsed
	}
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	if status != "" && status != "pending" && status != "delivering" && status != "retry" && status != "delivered" && status != "dead_letter" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_delivery_status"})
		return
	}
	rows, err := s.DB.Query(r.Context(), `SELECT d.id,d.event_id,e.event_type,d.target_id,t.name,d.status,d.attempt_count,d.next_attempt_at,d.last_http_status,d.last_error_code,d.delivered_at,d.created_at
		FROM webhook_deliveries d JOIN outbox_events e ON e.id=d.event_id JOIN webhook_targets t ON t.id=d.target_id
		WHERE ($1='' OR d.status=$1) ORDER BY d.created_at DESC,d.id DESC LIMIT $2`, status, limit)
	if err != nil {
		writeInternalError(w, r, "list webhook deliveries", err)
		return
	}
	defer rows.Close()
	deliveries := []webhookDeliveryView{}
	for rows.Next() {
		var d webhookDeliveryView
		if err := rows.Scan(&d.ID, &d.EventID, &d.EventType, &d.TargetID, &d.TargetName, &d.Status, &d.AttemptCount, &d.NextAttemptAt, &d.LastHTTPStatus, &d.LastErrorCode, &d.DeliveredAt, &d.CreatedAt); err != nil {
			writeInternalError(w, r, "scan webhook delivery", err)
			return
		}
		deliveries = append(deliveries, d)
	}
	if err := rows.Err(); err != nil {
		writeInternalError(w, r, "read webhook deliveries", err)
		return
	}
	writeJSON(w, http.StatusOK, deliveries)
}

func (s *Server) replayWebhookDelivery(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_webhook_delivery"})
		return
	}
	tx, err := s.DB.Begin(r.Context())
	if err != nil {
		writeInternalError(w, r, "begin webhook replay", err)
		return
	}
	defer tx.Rollback(r.Context())
	now := s.Now().UTC()
	command, err := tx.Exec(r.Context(), `UPDATE webhook_deliveries SET status='pending',attempt_count=0,next_attempt_at=$2,last_error_code=NULL,last_http_status=NULL,updated_at=$2 WHERE id=$1 AND status='dead_letter'`, id, now)
	if err != nil {
		writeInternalError(w, r, "replay webhook delivery", err)
		return
	}
	if command.RowsAffected() != 1 {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "delivery_not_dead_letter"})
		return
	}
	if _, err = tx.Exec(r.Context(), `UPDATE webhook_dead_letters SET replayed_at=$2 WHERE delivery_id=$1`, id, now); err != nil {
		writeInternalError(w, r, "mark webhook replay", err)
		return
	}
	if err = insertAdminAuditSuccess(r, db.New(tx), "webhook_delivery.replayed", "webhook_delivery", id.String(), "/v1/admin/webhook-deliveries/{id}/replay", http.StatusNoContent, struct {
		Mode string `json:"mode"`
	}{Mode: "manual"}); err != nil {
		writeInternalError(w, r, "audit webhook replay", err)
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		writeInternalError(w, r, "commit webhook replay", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

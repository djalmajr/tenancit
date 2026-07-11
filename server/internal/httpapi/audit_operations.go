package httpapi

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/djalmajr/tenancit/server/internal/auditops"
	"github.com/djalmajr/tenancit/server/internal/store/db"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
)

type createAuditExportRequest struct {
	Filters auditops.ExportFilter `json:"filters"`
	Format  string                `json:"format"`
}

func (s *Server) createAuditExport(w http.ResponseWriter, r *http.Request) {
	actor, ok := auditExportActor(r)
	if !ok || actor.Kind == "break_glass" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "insufficient permission"})
		return
	}
	key, err := uuid.Parse(strings.TrimSpace(r.Header.Get("Idempotency-Key")))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_idempotency_key"})
		return
	}
	var body createAuditExportRequest
	if !decodeJSON(w, r, &body) {
		return
	}
	if auditops.ValidateExport(body.Filters, body.Format) != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_audit_export"})
		return
	}
	count, err := s.AuditExports.Count(r.Context(), body.Filters)
	if err != nil {
		writeInternalError(w, r, "count audit export", err)
		return
	}
	if count > auditops.MaxExportRows {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "audit_export_too_large"})
		return
	}
	job, err := s.AuditExports.Create(
		r.Context(), actor, key, body.Filters, body.Format,
		count <= auditops.SyncExportLimit,
		auditops.DownloadAudit{
			RequestID: middleware.GetReqID(r.Context()),
			Method:    r.Method, RouteTemplate: "/v1/admin/audit-exports",
		},
	)
	if err != nil {
		if errors.Is(err, auditops.ErrIdempotencyMismatch) {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "idempotency_mismatch"})
			return
		}
		writeInternalError(w, r, "create audit export", err)
		return
	}
	status := http.StatusAccepted
	if job.Status == "ready" {
		status = http.StatusCreated
	}
	writeJSON(w, status, job)
}

func (s *Server) getAuditExport(w http.ResponseWriter, r *http.Request) {
	actor, ok := auditExportActor(r)
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if !ok || err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "audit_export_not_found"})
		return
	}
	job, err := s.AuditExports.Get(r.Context(), actor, id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "audit_export_not_found"})
		return
	}
	writeJSON(w, http.StatusOK, job)
}

func (s *Server) downloadAuditExport(w http.ResponseWriter, r *http.Request) {
	actor, ok := auditExportActor(r)
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if !ok || err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "audit_export_not_found"})
		return
	}
	payload, format, err := s.AuditExports.Consume(
		r.Context(), actor, id,
		auditops.DownloadAudit{
			RequestID: middleware.GetReqID(r.Context()),
			Method:    r.Method, RouteTemplate: "/v1/admin/audit-exports/{id}/download",
		},
	)
	if err != nil {
		status, code := http.StatusConflict, "audit_export_not_ready"
		if errors.Is(err, auditops.ErrExportExpired) {
			status, code = http.StatusGone, "audit_export_expired"
		}
		if errors.Is(err, auditops.ErrExportConsumed) {
			status, code = http.StatusGone, "audit_export_consumed"
		}
		writeJSON(w, status, map[string]string{"error": code})
		return
	}
	defer clear(payload)
	contentType, extension := "text/csv; charset=utf-8", "csv"
	if format == "jsonl" {
		contentType, extension = "application/x-ndjson", "jsonl"
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="tenancit-audit-%s.%s"`, id, extension))
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(payload)
}

func auditExportActor(r *http.Request) (auditops.ExportActor, bool) {
	value, ok := principalFromContext(r.Context())
	if !ok {
		return auditops.ExportActor{}, false
	}
	return auditops.ExportActor{Kind: string(value.Kind), Issuer: value.Issuer, Subject: value.Subject, Label: value.Label}, true
}

func (s *Server) auditHealth(w http.ResponseWriter, r *http.Request) {
	health, err := auditops.Health(r.Context(), s.DB, s.Now().UTC())
	if err != nil {
		writeInternalError(w, r, "load audit health", err)
		return
	}
	writeJSON(w, http.StatusOK, health)
}

type legalHoldRequest struct {
	From   time.Time `json:"from"`
	To     time.Time `json:"to"`
	Reason string    `json:"reason"`
}
type legalHoldAuditMetadata struct {
	From time.Time `json:"from"`
	To   time.Time `json:"to"`
}

type legalHoldView struct {
	ID                uuid.UUID  `json:"id"`
	From              time.Time  `json:"from"`
	To                time.Time  `json:"to"`
	Reason            string     `json:"reason"`
	CreatedByKind     string     `json:"created_by_kind"`
	CreatedByIssuer   *string    `json:"created_by_issuer,omitempty"`
	CreatedBySubject  string     `json:"created_by_subject"`
	CreatedAt         time.Time  `json:"created_at"`
	ReleasedAt        *time.Time `json:"released_at,omitempty"`
	ReleasedBySubject *string    `json:"released_by_subject,omitempty"`
}

func (s *Server) createAuditLegalHold(w http.ResponseWriter, r *http.Request) {
	actor, ok := principalFromContext(r.Context())
	var body legalHoldRequest
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "admin_auth_required"})
		return
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	if !body.To.After(body.From) || strings.TrimSpace(body.Reason) == "" || len(body.Reason) > 500 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_legal_hold"})
		return
	}
	tx, err := s.DB.Begin(r.Context())
	if err != nil {
		writeInternalError(w, r, "begin legal hold", err)
		return
	}
	defer tx.Rollback(r.Context())
	var id uuid.UUID
	err = tx.QueryRow(r.Context(), `INSERT INTO audit_legal_holds
		(from_time,to_time,reason,created_by_kind,created_by_issuer,created_by_subject)
		VALUES ($1,$2,$3,$4,NULLIF($5,''),$6) RETURNING id`,
		body.From, body.To, strings.TrimSpace(body.Reason), actor.Kind, actor.Issuer, actor.Subject,
	).Scan(&id)
	if err != nil {
		writeInternalError(w, r, "insert legal hold", err)
		return
	}
	err = insertAdminAuditSuccess(
		r, db.New(tx), "audit.legal_hold_created", "audit_legal_hold", id.String(),
		"/v1/admin/audit-legal-holds", http.StatusCreated,
		legalHoldAuditMetadata{From: body.From, To: body.To},
	)
	if err != nil {
		writeInternalError(w, r, "audit legal hold", err)
		return
	}
	err = tx.Commit(r.Context())
	if err != nil {
		writeInternalError(w, r, "create legal hold", err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"id": id})
}

func (s *Server) listAuditLegalHolds(w http.ResponseWriter, r *http.Request) {
	rows, err := s.DB.Query(r.Context(), `SELECT
		id,from_time,to_time,reason,created_by_kind,created_by_issuer,
		created_by_subject,created_at,released_at,released_by_subject
		FROM audit_legal_holds ORDER BY created_at DESC LIMIT 200`)
	if err != nil {
		writeInternalError(w, r, "list legal holds", err)
		return
	}
	defer rows.Close()
	items := []legalHoldView{}
	for rows.Next() {
		var item legalHoldView
		if err := rows.Scan(
			&item.ID, &item.From, &item.To, &item.Reason, &item.CreatedByKind,
			&item.CreatedByIssuer, &item.CreatedBySubject, &item.CreatedAt,
			&item.ReleasedAt, &item.ReleasedBySubject,
		); err != nil {
			writeInternalError(w, r, "scan legal hold", err)
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		writeInternalError(w, r, "iterate legal holds", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) releaseAuditLegalHold(w http.ResponseWriter, r *http.Request) {
	actor, ok := principalFromContext(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if !ok || err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "legal_hold_not_found"})
		return
	}
	tx, err := s.DB.Begin(r.Context())
	if err != nil {
		writeInternalError(w, r, "begin release legal hold", err)
		return
	}
	defer tx.Rollback(r.Context())
	tag, err := tx.Exec(r.Context(), `UPDATE audit_legal_holds
		SET released_at=clock_timestamp(),released_by_subject=$2
		WHERE id=$1 AND released_at IS NULL`, id, actor.Subject)
	if err == nil && tag.RowsAffected() != 1 {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "legal_hold_not_active"})
		return
	}
	if err != nil {
		writeInternalError(w, r, "release legal hold", err)
		return
	}
	err = insertAdminAuditSuccess(
		r, db.New(tx), "audit.legal_hold_released", "audit_legal_hold", id.String(),
		"/v1/admin/audit-legal-holds/{id}/release", http.StatusNoContent, struct{}{},
	)
	if err != nil {
		writeInternalError(w, r, "audit legal hold release", err)
		return
	}
	err = tx.Commit(r.Context())
	if err != nil {
		writeInternalError(w, r, "release legal hold", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

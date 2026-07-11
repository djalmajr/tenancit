package httpapi

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/djalmajr/tenancit/server/internal/service"
	"github.com/djalmajr/tenancit/server/internal/telemetry"
	"github.com/jackc/pgx/v5"
)

var operationalIdentifier = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`)

type operationalReportBody struct {
	Kind            string    `json:"kind"`
	Source          string    `json:"source"`
	Status          string    `json:"status"`
	OccurredAt      time.Time `json:"occurred_at"`
	FreshForSeconds int       `json:"fresh_for_seconds"`
}

type operationalReportView struct {
	ID                string    `json:"id"`
	Kind              string    `json:"kind"`
	Source            string    `json:"source"`
	Status            string    `json:"status"`
	EffectiveStatus   string    `json:"effective_status"`
	OccurredAt        time.Time `json:"occurred_at"`
	FreshUntil        time.Time `json:"fresh_until"`
	ReceivedAt        time.Time `json:"received_at"`
	CredentialVersion string    `json:"credential_version"`
}

type operationalQueueView struct {
	WebhookPending    int64 `json:"webhook_pending"`
	WebhookRetry      int64 `json:"webhook_retry"`
	WebhookDeadLetter int64 `json:"webhook_dead_letter"`
	OpenCircuits      int64 `json:"open_circuits"`
}

type operationalHealthView struct {
	Status     string                      `json:"status"`
	CheckedAt  time.Time                   `json:"checked_at"`
	Components []telemetry.ComponentStatus `json:"components"`
	Reports    []operationalReportView     `json:"reports"`
	Queues     operationalQueueView        `json:"queues"`
}

func (s *Server) SetOperationsReportCredential(token, version string) {
	token = strings.TrimSpace(token)
	version = strings.TrimSpace(version)
	if token == "" || version == "" {
		s.OperationsReportTokenHash = ""
		s.OperationsReportCredentialVersion = ""
		return
	}
	s.OperationsReportTokenHash = service.HashAPIKey(token)
	s.OperationsReportCredentialVersion = version
}

func (s *Server) requireOperationsReporter(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.OperationsReportTokenHash == "" {
			telemetry.RecordSecurityDecision(r.Context(), "operations_report", "unavailable")
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "operations_reporting_unavailable"})
			return
		}
		if !constantTimeCredentialMatch(s.OperationsReportTokenHash, bearerToken(r)) {
			telemetry.RecordSecurityDecision(r.Context(), "operations_report", "denied")
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid_operations_credential"})
			return
		}
		telemetry.RecordSecurityDecision(r.Context(), "operations_report", "success")
		next.ServeHTTP(w, r)
	})
}

func (s *Server) createOperationalReport(w http.ResponseWriter, r *http.Request) {
	idempotencyKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	var body operationalReportBody
	if !decodeJSON(w, r, &body) {
		return
	}
	body.Kind = strings.TrimSpace(body.Kind)
	body.Source = strings.TrimSpace(body.Source)
	body.Status = strings.TrimSpace(body.Status)
	body.OccurredAt = body.OccurredAt.UTC()
	now := s.Now().UTC()
	if !validOperationalReport(body, idempotencyKey, now) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_operational_report"})
		return
	}
	freshUntil := body.OccurredAt.UTC().Add(time.Duration(body.FreshForSeconds) * time.Second)
	payload, _ := json.Marshal(body)
	payloadHash := sha256.Sum256(payload)
	var report operationalReportView
	err := s.DB.QueryRow(r.Context(), `INSERT INTO operational_reports
		(kind,source,status,occurred_at,fresh_until,idempotency_key,payload_hash,credential_version)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		ON CONFLICT (source,idempotency_key) DO NOTHING
		RETURNING id,kind,source,status,occurred_at,fresh_until,received_at,credential_version`,
		body.Kind, body.Source, body.Status, body.OccurredAt.UTC(), freshUntil,
		idempotencyKey, payloadHash[:], s.OperationsReportCredentialVersion,
	).Scan(&report.ID, &report.Kind, &report.Source, &report.Status, &report.OccurredAt, &report.FreshUntil, &report.ReceivedAt, &report.CredentialVersion)
	status := http.StatusCreated
	if errors.Is(err, pgx.ErrNoRows) {
		var existingHash []byte
		err = s.DB.QueryRow(r.Context(), `SELECT id,kind,source,status,occurred_at,fresh_until,received_at,credential_version,payload_hash
			FROM operational_reports WHERE source=$1 AND idempotency_key=$2`, body.Source, idempotencyKey).
			Scan(&report.ID, &report.Kind, &report.Source, &report.Status, &report.OccurredAt, &report.FreshUntil, &report.ReceivedAt, &report.CredentialVersion, &existingHash)
		if err == nil && !bytes.Equal(existingHash, payloadHash[:]) {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "idempotency_conflict"})
			return
		}
		status = http.StatusOK
	}
	if err != nil {
		writeInternalError(w, r, "persist operational report", err)
		return
	}
	report.EffectiveStatus = effectiveReportStatus(report.Status, report.FreshUntil, now)
	w.Header().Set("Cache-Control", "private, no-store")
	writeJSON(w, status, report)
}

func validOperationalReport(body operationalReportBody, idempotencyKey string, now time.Time) bool {
	validKind := body.Kind == "backup" || body.Kind == "restore" || body.Kind == "rewrap" || body.Kind == "migration"
	validStatus := body.Status == "healthy" || body.Status == "degraded" || body.Status == "failed"
	validTime := !body.OccurredAt.IsZero() && !body.OccurredAt.After(now.Add(5*time.Minute)) && !body.OccurredAt.Before(now.AddDate(0, 0, -30))
	return validKind && validStatus && operationalIdentifier.MatchString(body.Source) && operationalIdentifier.MatchString(idempotencyKey) &&
		body.FreshForSeconds >= 60 && body.FreshForSeconds <= 7*24*60*60 && validTime
}

func effectiveReportStatus(status string, freshUntil, now time.Time) string {
	if !now.Before(freshUntil) {
		return "stale"
	}
	return status
}

func (s *Server) operationalHealth(w http.ResponseWriter, r *http.Request) {
	readiness := telemetry.EvaluateReadiness(r.Context(), s.ReadinessProbes, s.Now)
	now := s.Now().UTC()
	rows, err := s.DB.Query(r.Context(), `SELECT DISTINCT ON (kind,source)
		id,kind,source,status,occurred_at,fresh_until,received_at,credential_version
		FROM operational_reports ORDER BY kind,source,occurred_at DESC,id DESC`)
	if err != nil {
		writeInternalError(w, r, "list operational reports", err)
		return
	}
	defer rows.Close()
	reports := []operationalReportView{}
	status := string(readiness.Status)
	for rows.Next() {
		var report operationalReportView
		if err := rows.Scan(&report.ID, &report.Kind, &report.Source, &report.Status, &report.OccurredAt, &report.FreshUntil, &report.ReceivedAt, &report.CredentialVersion); err != nil {
			writeInternalError(w, r, "scan operational report", err)
			return
		}
		report.EffectiveStatus = effectiveReportStatus(report.Status, report.FreshUntil, now)
		if report.EffectiveStatus != "healthy" && status == string(telemetry.StatusHealthy) {
			status = string(telemetry.StatusDegraded)
		}
		reports = append(reports, report)
	}
	var queues operationalQueueView
	if err := s.DB.QueryRow(r.Context(), `SELECT
		count(*) FILTER(WHERE status IN ('pending','delivering')),
		count(*) FILTER(WHERE status='retry'),count(*) FILTER(WHERE status='dead_letter'),
		(SELECT count(*) FROM webhook_targets WHERE circuit_open_until>clock_timestamp())
		FROM webhook_deliveries`).Scan(&queues.WebhookPending, &queues.WebhookRetry, &queues.WebhookDeadLetter, &queues.OpenCircuits); err != nil {
		writeInternalError(w, r, "load operational queues", err)
		return
	}
	if (queues.WebhookDeadLetter > 0 || queues.OpenCircuits > 0) && status == string(telemetry.StatusHealthy) {
		status = string(telemetry.StatusDegraded)
	}
	w.Header().Set("Cache-Control", "private, no-store")
	writeJSON(w, http.StatusOK, operationalHealthView{
		Status: status, CheckedAt: now, Components: readiness.Components, Reports: reports, Queues: queues,
	})
}

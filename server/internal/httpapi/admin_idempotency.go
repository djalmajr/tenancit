package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/djalmajr/tenancit/server/internal/idempotency"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const (
	adminMutationTTL = 24 * time.Hour
	adminSecretTTL   = 10 * time.Minute
)

func prepareAdminIdempotency(w http.ResponseWriter, r *http.Request, operation string, payload any, ttl time.Duration) (idempotency.Request, bool) {
	key, err := uuid.Parse(r.Header.Get("Idempotency-Key"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "idempotency_key_required"})
		return idempotency.Request{}, false
	}
	principal, ok := principalFromContext(r.Context())
	if !ok || (principal.Kind != principalKindSharedAdminToken && principal.Kind != principalKindOIDCUser) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "idempotency_not_available"})
		return idempotency.Request{}, false
	}
	canonical, err := json.Marshal(payload)
	if err != nil {
		writeInternalError(w, r, "encode idempotency fingerprint", err)
		return idempotency.Request{}, false
	}
	request := idempotency.Request{
		Actor:     idempotency.Actor{Kind: string(principal.Kind), Issuer: principal.Issuer, Subject: principal.Subject},
		Operation: operation, Key: key, Fingerprint: idempotency.Fingerprint(operation, canonical), TTL: ttl,
	}
	clear(canonical)
	return request, true
}

func (s *Server) beginAdminIdempotency(w http.ResponseWriter, r *http.Request, tx pgx.Tx, request idempotency.Request) (bool, bool) {
	replay, err := s.Idempotency.Begin(r.Context(), tx, request)
	if err != nil {
		switch {
		case errors.Is(err, idempotency.ErrFingerprintMismatch):
			writeJSON(w, http.StatusConflict, map[string]string{"error": "idempotency_mismatch"})
		case errors.Is(err, idempotency.ErrInProgress):
			w.Header().Set("Retry-After", "1")
			writeJSON(w, http.StatusConflict, map[string]string{"error": "idempotency_in_progress"})
		case errors.Is(err, idempotency.ErrExpired):
			writeJSON(w, http.StatusConflict, map[string]string{"error": "idempotency_expired"})
		default:
			writeInternalError(w, r, "claim idempotency key", err)
		}
		return false, false
	}
	if !replay.Found {
		return true, false
	}
	defer clear(replay.Body)
	if replay.ContentType != "" {
		w.Header().Set("Content-Type", replay.ContentType)
	}
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("Idempotency-Replayed", "true")
	w.WriteHeader(replay.Status)
	_, _ = w.Write(replay.Body)
	return false, true
}

func encodeIdempotentResponse(value any) ([]byte, error) {
	return json.Marshal(value)
}

func (s *Server) completeAdminIdempotency(r *http.Request, tx pgx.Tx, request idempotency.Request, status int, body []byte) error {
	return s.Idempotency.Complete(r.Context(), tx, request, status, "application/json; charset=utf-8", body)
}

func writeIdempotentResponse(w http.ResponseWriter, status int, body []byte, secret bool) {
	if secret {
		w.Header().Set("Cache-Control", "private, no-store")
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

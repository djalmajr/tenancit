package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/djalmajr/tenancit/server/internal/telemetry"
)

// Health responds with the service liveness status.
func Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Readiness checks runtime dependencies synchronously and returns only stable,
// low-cardinality component names and statuses. Probe errors never cross the
// HTTP boundary because they may contain DSNs, URLs, or credentials.
func (s *Server) Readiness(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	report := telemetry.EvaluateReadiness(ctx, s.ReadinessProbes, s.Now)
	status := http.StatusOK
	if report.Status == telemetry.StatusUnavailable {
		status = http.StatusServiceUnavailable
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, status, report)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

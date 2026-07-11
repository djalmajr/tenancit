package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/djalmajr/tenancit/server/internal/ratelimit"
	"github.com/djalmajr/tenancit/server/internal/telemetry"
	usageevents "github.com/djalmajr/tenancit/server/internal/usage"
	"github.com/google/uuid"
)

func EnforceAPIClientRateLimit(
	limiter ratelimit.Limiter,
	recorder usageRecorder,
	operation string,
	now func() time.Time,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			principal, ok := apiClientPrincipalFromContext(r.Context())
			if !ok || principal.RPMLimit == nil {
				next.ServeHTTP(w, r)
				return
			}
			result, err := limiter.Allow(r.Context(), principal.ID, *principal.RPMLimit)
			if err != nil {
				if errors.Is(err, ratelimit.ErrUnavailable) {
					telemetry.RecordSecurityDecision(r.Context(), "rate_limit", "unavailable")
					writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "rate_limiter_unavailable"})
					return
				}
				writeInternalError(w, r, "check API client rate limit", err)
				return
			}
			w.Header().Set("RateLimit-Limit", strconv.Itoa(int(result.Limit)))
			w.Header().Set("RateLimit-Remaining", strconv.Itoa(int(result.Remaining)))
			w.Header().Set("RateLimit-Reset", strconv.FormatInt(ceilSeconds(result.ResetAfter), 10))
			if !result.Allowed {
				telemetry.RecordSecurityDecision(r.Context(), "rate_limit", "limited")
				retry := ceilSeconds(result.RetryAfter)
				w.Header().Set("Retry-After", strconv.FormatInt(retry, 10))
				if clientID, parseErr := uuid.Parse(principal.ID); parseErr == nil {
					recorder.Record(usageevents.Event{
						APIClientID: clientID, Operation: operation, Status: http.StatusTooManyRequests,
						RateLimited: true, At: now().UTC(),
					})
				}
				writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "rate_limited"})
				return
			}
			telemetry.RecordSecurityDecision(r.Context(), "rate_limit", "allowed")
			next.ServeHTTP(w, r)
		})
	}
}

func ceilSeconds(value time.Duration) int64 {
	seconds := int64((value + time.Second - 1) / time.Second)
	if seconds < 1 {
		return 1
	}
	return seconds
}

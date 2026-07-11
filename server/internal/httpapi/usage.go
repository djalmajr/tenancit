package httpapi

import (
	"net/http"
	"time"

	usageevents "github.com/djalmajr/tenancit/server/internal/usage"
	"github.com/google/uuid"
)

type usageRecorder interface {
	Record(usageevents.Event)
}

type discardUsageRecorder struct{}

func (discardUsageRecorder) Record(usageevents.Event) {}

type responseStatusWriter struct {
	http.ResponseWriter
	status int
}

func (w *responseStatusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *responseStatusWriter) Write(body []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(body)
}

func RecordAPIUsage(recorder usageRecorder, operation string, now func() time.Time) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			principal, ok := apiClientPrincipalFromContext(r.Context())
			if !ok {
				next.ServeHTTP(w, r)
				return
			}
			clientID, err := uuid.Parse(principal.ID)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}
			wrapped := &responseStatusWriter{ResponseWriter: w}
			next.ServeHTTP(wrapped, r)
			status := wrapped.status
			if status == 0 {
				status = http.StatusOK
			}
			recorder.Record(usageevents.Event{
				APIClientID: clientID, Operation: operation, Status: status, At: now().UTC(),
			})
		})
	}
}

package httpapi

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	publicInternalError = "internal error"
	publicNotFound      = "not found"
)

func writeNotFound(w http.ResponseWriter) {
	writeJSON(w, http.StatusNotFound, map[string]string{"error": publicNotFound})
}

// writeInternalError keeps implementation details out of the response. Logs
// intentionally omit request headers, body, PostgreSQL Detail and error text:
// those fields may contain credentials or tenant data.
func writeInternalError(w http.ResponseWriter, r *http.Request, operation string, err error) {
	attrs := []any{
		"operation", operation,
		"request_id", middleware.GetReqID(r.Context()),
		"error_type", fmt.Sprintf("%T", err),
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		attrs = append(attrs, "postgres_code", pgErr.Code, "constraint", pgErr.ConstraintName)
	}
	slog.ErrorContext(r.Context(), "request failed", attrs...)
	writeJSON(w, http.StatusInternalServerError, map[string]string{"error": publicInternalError})
}

func isNotFound(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}

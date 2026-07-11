package httpapi

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/djalmajr/tenancit/server/internal/service"
	"github.com/djalmajr/tenancit/server/internal/store/db"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// AuditAdminFailures records denied and failed admin requests without changing
// their outcome. Successful mutations remain fail-closed and transactional in
// their handlers; failures are necessarily best-effort because no domain
// transaction may exist to join.
func (s *Server) AuditAdminFailures(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wrapped := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(wrapped, r)
		status := wrapped.Status()
		if status < http.StatusBadRequest {
			return
		}
		actorKind, actorSubject := "unauthenticated", "anonymous"
		token := bearerToken(r)
		if token != "" {
			hash := service.HashAPIKey(token)
			if subtle.ConstantTimeCompare([]byte(hash), []byte(s.AdminTokenHash)) == 1 {
				actorKind, actorSubject = string(principalKindSharedAdminToken), "primary"
			}
		}
		result, action := "error", "admin.request_failed"
		if status == http.StatusUnauthorized || status == http.StatusForbidden {
			result, action = "denied", "admin.request_denied"
		}
		route := chi.RouteContext(r.Context()).RoutePattern()
		if route == "" {
			route = "/v1/admin/*"
		}
		_, err := s.Q.InsertAdminAuditEvent(r.Context(), db.InsertAdminAuditEventParams{
			RequestID: middleware.GetReqID(r.Context()), ActorKind: actorKind, ActorSubject: actorSubject,
			Action: action, TargetType: "admin_route", TargetID: route, Result: result,
			HttpMethod: r.Method, RouteTemplate: route, HttpStatus: int16(status), Metadata: []byte(`{}`),
		})
		if err != nil {
			slog.Error("admin failure audit write failed", "request_id", middleware.GetReqID(r.Context()), "status", status, "error_type", fmt.Sprintf("%T", err))
		}
	})
}

func insertAdminAuditSuccess(
	r *http.Request,
	q *db.Queries,
	action string,
	targetType string,
	targetID string,
	routeTemplate string,
	status int,
	metadataValue any,
) error {
	actor, ok := principalFromContext(r.Context())
	if !ok {
		return errors.New("admin principal missing")
	}
	metadata, err := json.Marshal(metadataValue)
	if err != nil {
		return err
	}
	_, err = q.InsertAdminAuditEvent(r.Context(), db.InsertAdminAuditEventParams{
		RequestID: middleware.GetReqID(r.Context()),
		ActorKind: string(actor.Kind), ActorSubject: actor.Subject,
		Action: action, TargetType: targetType, TargetID: targetID,
		Result: "success", HttpMethod: r.Method, RouteTemplate: routeTemplate,
		HttpStatus: int16(status), Metadata: metadata,
	})
	return err
}

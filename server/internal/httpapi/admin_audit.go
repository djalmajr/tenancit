package httpapi

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/djalmajr/tenancit/server/internal/events"
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
		ctx, capture := contextWithPrincipalCapture(r.Context())
		r = r.WithContext(ctx)
		wrapped := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(wrapped, r)
		status := wrapped.Status()
		if status == 0 {
			status = http.StatusOK
		}
		if status < http.StatusBadRequest {
			if capture.ok && capture.value.Kind == principalKindBreakGlass {
				route := chi.RouteContext(r.Context()).RoutePattern()
				if route == "" {
					route = "/v1/admin/*"
				}
				_, err := s.Q.InsertAdminAuditEvent(r.Context(), db.InsertAdminAuditEventParams{
					RequestID: middleware.GetReqID(r.Context()), ActorKind: string(capture.value.Kind),
					ActorSubject: capture.value.Subject, Action: "break_glass.request_succeeded",
					TargetType: "admin_route", TargetID: route, Result: "success",
					HttpMethod: r.Method, RouteTemplate: route, HttpStatus: int16(status), Metadata: []byte(`{}`),
				})
				if err != nil {
					slog.Error("break-glass audit write failed", "request_id", middleware.GetReqID(r.Context()), "status", status, "error_type", fmt.Sprintf("%T", err))
				}
			}
			return
		}
		actorKind, actorSubject := "unauthenticated", "anonymous"
		var actorIssuer, actorLabel *string
		if capture.ok {
			actorKind, actorSubject = string(capture.value.Kind), capture.value.Subject
			actorIssuer, actorLabel = optionalString(capture.value.Issuer), optionalString(capture.value.Label)
		} else if token := bearerToken(r); token != "" {
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
			ActorIssuer: actorIssuer, ActorLabel: actorLabel,
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
		RequestID: middleware.GetReqID(r.Context()), ActorKind: string(actor.Kind),
		ActorIssuer: optionalString(actor.Issuer), ActorSubject: actor.Subject,
		ActorLabel: optionalString(actor.Label),
		Action:     action, TargetType: targetType, TargetID: targetID,
		Result: "success", HttpMethod: r.Method, RouteTemplate: routeTemplate,
		HttpStatus: int16(status), Metadata: metadata,
	})
	if err != nil {
		return err
	}
	draft, publish, err := events.FromAudit(action, targetType, targetID)
	if err != nil || !publish {
		return err
	}
	event, err := q.InsertOutboxEvent(r.Context(), db.InsertOutboxEventParams{
		EventType: draft.Type, EventVersion: draft.Version,
		AggregateType: draft.AggregateType, AggregateID: draft.AggregateID,
		RequestID: middleware.GetReqID(r.Context()), Payload: draft.Payload,
	})
	if err != nil {
		return err
	}
	return q.EnqueueOutboxDeliveries(r.Context(), event.ID)
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

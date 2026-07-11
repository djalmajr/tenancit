package httpapi

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/djalmajr/tenancit/server/internal/telemetry"
)

type adminPermission string

const (
	permissionAdminRead         adminPermission = "admin.read"
	permissionAuditRead         adminPermission = "audit.read"
	permissionAPIClientManage   adminPermission = "api_client.manage"
	permissionIntegrationManage adminPermission = "integration.manage"
	permissionResourceWrite     adminPermission = "resource.write"
	permissionSecretReveal      adminPermission = "secret.reveal"
	permissionSessionManage     adminPermission = "session.manage"
	permissionSettingsManage    adminPermission = "settings.manage"
	permissionTenantHardDelete  adminPermission = "tenant.hard_delete"
	permissionTenantWrite       adminPermission = "tenant.write"
)

var sharedAdminPermissions = [...]adminPermission{
	permissionAdminRead,
	permissionAuditRead,
	permissionAPIClientManage,
	permissionIntegrationManage,
	permissionResourceWrite,
	permissionSecretReveal,
	permissionSessionManage,
	permissionSettingsManage,
	permissionTenantHardDelete,
	permissionTenantWrite,
}

type principalKind string

const (
	principalKindSharedAdminToken principalKind = "shared_admin_token"
	principalKindBreakGlass       principalKind = "break_glass"
	principalKindOIDCUser         principalKind = "oidc_user"
)

type adminRole string

const (
	roleViewer        adminRole = "viewer"
	roleOperator      adminRole = "operator"
	roleSecurityAdmin adminRole = "security_admin"
)

var permissionsByRole = map[adminRole][]adminPermission{
	roleViewer: {
		permissionAdminRead,
	},
	roleOperator: {
		permissionAdminRead,
		permissionTenantWrite,
		permissionResourceWrite,
		permissionAPIClientManage,
		permissionIntegrationManage,
	},
	roleSecurityAdmin: {
		permissionAdminRead,
		permissionAuditRead,
		permissionAPIClientManage,
		permissionIntegrationManage,
		permissionResourceWrite,
		permissionSecretReveal,
		permissionSessionManage,
		permissionSettingsManage,
		permissionTenantHardDelete,
		permissionTenantWrite,
	},
}

type principal struct {
	Kind          principalKind
	Issuer        string
	Subject       string
	Label         string
	SessionID     string
	Roles         []adminRole
	csrfTokenHash string
	permissions   map[adminPermission]struct{}
}

type principalContextKey struct{}
type principalCaptureContextKey struct{}

type principalCapture struct {
	value principal
	ok    bool
}

func newPrincipal(kind principalKind, subject string, permissions ...adminPermission) principal {
	permissionSet := make(map[adminPermission]struct{}, len(permissions))
	for _, permission := range permissions {
		permissionSet[permission] = struct{}{}
	}
	return principal{
		Kind:        kind,
		Subject:     subject,
		permissions: permissionSet,
	}
}

func newOIDCPrincipal(issuer, subject, label, sessionID string, roles []adminRole) (principal, error) {
	issuer = strings.TrimSpace(issuer)
	subject = strings.TrimSpace(subject)
	sessionID = strings.TrimSpace(sessionID)
	if issuer == "" || subject == "" || sessionID == "" || len(roles) == 0 {
		return principal{}, errors.New("OIDC principal identity, session, and roles are required")
	}
	permissions := make([]adminPermission, 0, len(sharedAdminPermissions))
	seenRoles := make(map[adminRole]struct{}, len(roles))
	normalizedRoles := make([]adminRole, 0, len(roles))
	for _, role := range roles {
		rolePermissions, ok := permissionsByRole[role]
		if !ok {
			return principal{}, errors.New("OIDC principal contains an unknown role")
		}
		if _, duplicate := seenRoles[role]; duplicate {
			continue
		}
		seenRoles[role] = struct{}{}
		normalizedRoles = append(normalizedRoles, role)
		permissions = append(permissions, rolePermissions...)
	}
	value := newPrincipal(principalKindOIDCUser, subject, permissions...)
	value.Issuer = issuer
	value.Label = strings.TrimSpace(label)
	value.SessionID = sessionID
	value.Roles = normalizedRoles
	return value, nil
}

func newBreakGlassPrincipal(version string) principal {
	return newPrincipal(
		principalKindBreakGlass,
		"admin-token:"+strings.TrimSpace(version),
		permissionAdminRead,
		permissionAuditRead,
	)
}

func (p principal) hasPermission(permission adminPermission) bool {
	_, ok := p.permissions[permission]
	return ok
}

func contextWithPrincipal(ctx context.Context, value principal) context.Context {
	if capture, ok := ctx.Value(principalCaptureContextKey{}).(*principalCapture); ok && capture != nil {
		capture.value = value
		capture.ok = true
	}
	return context.WithValue(ctx, principalContextKey{}, value)
}

func contextWithPrincipalCapture(ctx context.Context) (context.Context, *principalCapture) {
	capture := &principalCapture{}
	return context.WithValue(ctx, principalCaptureContextKey{}, capture), capture
}

func principalFromContext(ctx context.Context) (principal, bool) {
	value, ok := ctx.Value(principalContextKey{}).(principal)
	return value, ok
}

func requireAdminPermission(permission adminPermission) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			value, ok := principalFromContext(r.Context())
			if !ok {
				telemetry.RecordSecurityDecision(r.Context(), "admin_permission", "denied")
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid admin token"})
				return
			}
			if !value.hasPermission(permission) {
				telemetry.RecordSecurityDecision(r.Context(), "admin_permission", "denied")
				writeJSON(w, http.StatusForbidden, map[string]string{"error": "insufficient permission"})
				return
			}
			telemetry.RecordSecurityDecision(r.Context(), "admin_permission", "success")
			next.ServeHTTP(w, r)
		})
	}
}

func requireSecretRevealPermission(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("reveal") != "true" {
			next.ServeHTTP(w, r)
			return
		}
		requireAdminPermission(permissionSecretReveal)(next).ServeHTTP(w, r)
	})
}

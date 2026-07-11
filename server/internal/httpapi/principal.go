package httpapi

import (
	"context"
	"net/http"
)

type adminPermission string

const (
	permissionAdminRead        adminPermission = "admin.read"
	permissionAuditRead        adminPermission = "audit.read"
	permissionAPIClientManage  adminPermission = "api_client.manage"
	permissionResourceWrite    adminPermission = "resource.write"
	permissionSecretReveal     adminPermission = "secret.reveal"
	permissionTenantHardDelete adminPermission = "tenant.hard_delete"
	permissionTenantWrite      adminPermission = "tenant.write"
)

var sharedAdminPermissions = [...]adminPermission{
	permissionAdminRead,
	permissionAuditRead,
	permissionAPIClientManage,
	permissionResourceWrite,
	permissionSecretReveal,
	permissionTenantHardDelete,
	permissionTenantWrite,
}

type principalKind string

const principalKindSharedAdminToken principalKind = "shared_admin_token"

type principal struct {
	Kind        principalKind
	Subject     string
	permissions map[adminPermission]struct{}
}

type principalContextKey struct{}

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

func (p principal) hasPermission(permission adminPermission) bool {
	_, ok := p.permissions[permission]
	return ok
}

func contextWithPrincipal(ctx context.Context, value principal) context.Context {
	return context.WithValue(ctx, principalContextKey{}, value)
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
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid admin token"})
				return
			}
			if !value.hasPermission(permission) {
				writeJSON(w, http.StatusForbidden, map[string]string{"error": "insufficient permission"})
				return
			}
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

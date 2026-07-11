package httpapi

import (
	"net/http"
	"time"

	"github.com/djalmajr/tenancit/server/internal/adminauth"
	"github.com/djalmajr/tenancit/server/internal/crypto"
	"github.com/djalmajr/tenancit/server/internal/ratelimit"
	"github.com/djalmajr/tenancit/server/internal/service"
	"github.com/djalmajr/tenancit/server/internal/store/db"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Server holds dependencies shared by handlers.
type Server struct {
	AdminTokenHash string
	Cryptor        *crypto.Cryptor
	DB             *pgxpool.Pool
	Q              *db.Queries
	Resolver       *service.Resolver
	Now            func() time.Time
	Usage          usageRecorder
	Limiter        ratelimit.Limiter
	AdminAuth      *AdminAuthRuntime
}

// NewServer wires a Server from the database pool, cryptor, and admin token.
func NewServer(pool *pgxpool.Pool, c *crypto.Cryptor, adminToken string) *Server {
	q := db.New(pool)
	return &Server{
		AdminTokenHash: service.HashAPIKey(adminToken),
		Cryptor:        c,
		DB:             pool,
		Q:              q,
		Resolver:       service.NewResolver(q, c),
		Now:            time.Now,
		Usage:          discardUsageRecorder{},
		Limiter:        ratelimit.NewMemory(time.Now),
	}
}

// Routes builds the full router: health, consumer API (api-key), admin API,
// and the embedded SPA fallback.
func (s *Server) Routes(staticHandler http.Handler) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(SecurityHeaders)
	r.Use(middleware.Recoverer)

	r.Get("/healthz", Health)

	if s.AdminAuth != nil && s.AdminAuth.Config.Mode == adminauth.ModeOIDC {
		r.Get("/v1/auth/login", s.startOIDCLogin)
		r.Get("/v1/auth/callback", s.completeOIDCLogin)
		r.With(RequireAdminSession(s.AdminAuth.Sessions, s.AdminAuth.Config.CookieName)).Get("/v1/auth/session", s.getAdminSession)
		r.With(
			RequireAdminSession(s.AdminAuth.Sessions, s.AdminAuth.Config.CookieName),
			RequireAdminCSRF(s.AdminAuth.Config.AdminOrigin),
		).Post("/v1/auth/logout", s.logoutAdminSession)
	}

	r.Group(func(cr chi.Router) {
		cr.Use(RequireAPIKey(s.Q, s.Now))
		cr.With(RequireAPIClientScope(service.ScopeTenantIdentify), EnforceAPIClientRateLimit(s.Limiter, s.Usage, "identify", s.Now), RecordAPIUsage(s.Usage, "identify", s.Now)).Get("/v1/identify", s.handleIdentify)
		cr.With(RequireAPIClientScope(service.ScopeResourceResolve), EnforceAPIClientRateLimit(s.Limiter, s.Usage, "resolve", s.Now), RecordAPIUsage(s.Usage, "resolve", s.Now)).Get("/v1/resolve", s.handleResolve)
		cr.With(RequireAPIClientScope(service.ScopeResourceResolve), EnforceAPIClientRateLimit(s.Limiter, s.Usage, "resolve", s.Now), RecordAPIUsage(s.Usage, "resolve", s.Now)).Get("/v1/resolve/{hostname}/resources/{definitionKey}", s.handleResolveOne)
	})

	r.Route("/v1/admin", func(ar chi.Router) {
		ar.Use(s.AuditAdminFailures)
		if s.AdminAuth != nil && s.AdminAuth.Config.Mode == adminauth.ModeOIDC {
			ar.Use(RequireAdminSession(s.AdminAuth.Sessions, s.AdminAuth.Config.CookieName))
			ar.Use(RequireAdminCSRF(s.AdminAuth.Config.AdminOrigin))
		} else {
			ar.Use(RequireAdminToken(s.AdminTokenHash))
		}

		ar.With(requireAdminPermission(permissionAdminRead)).Get("/overview", s.overview)
		ar.With(requireAdminPermission(permissionAuditRead)).Get("/audit-events", s.listAuditEvents)

		ar.With(requireAdminPermission(permissionTenantWrite)).Post("/tenants", s.createTenant)
		ar.With(requireAdminPermission(permissionAdminRead)).Get("/tenants", s.listTenants)
		ar.With(requireAdminPermission(permissionAdminRead)).Get("/tenants/{id}", s.getTenant)
		ar.With(requireAdminPermission(permissionTenantWrite)).Put("/tenants/{id}", s.updateTenant)
		ar.With(requireAdminPermission(permissionTenantHardDelete)).Delete("/tenants/{id}", s.deleteTenant)
		ar.With(requireAdminPermission(permissionTenantWrite)).Post("/tenants/{id}/domains", s.addDomain)
		ar.With(requireAdminPermission(permissionAdminRead)).Get("/tenants/{id}/domains", s.listTenantDomains)
		ar.With(requireAdminPermission(permissionTenantWrite)).Delete("/tenants/{id}/domains/{domainId}", s.deleteDomain)
		ar.With(requireAdminPermission(permissionResourceWrite)).Post("/tenants/{id}/resources", s.createResource)
		ar.With(
			requireAdminPermission(permissionAdminRead),
			requireSecretRevealPermission,
		).Get("/tenants/{id}/resources", s.listTenantResources)
		ar.With(requireAdminPermission(permissionResourceWrite)).Put("/tenants/{id}/resources/{resourceId}/status", s.setResourceStatus)
		ar.With(requireAdminPermission(permissionResourceWrite)).Delete("/tenants/{id}/resources/{resourceId}", s.deleteResource)

		ar.With(requireAdminPermission(permissionResourceWrite)).Post("/resource-definitions", s.createDefinition)
		ar.With(requireAdminPermission(permissionAdminRead)).Get("/resource-definitions", s.listDefinitions)
		ar.With(requireAdminPermission(permissionAdminRead)).Get("/resource-definitions/{id}", s.getDefinition)
		ar.With(requireAdminPermission(permissionResourceWrite)).Put("/resource-definitions/{id}/status", s.setDefinitionStatus)
		ar.With(requireAdminPermission(permissionResourceWrite)).Post("/resource-definitions/{id}/fields", s.addField)
		ar.With(requireAdminPermission(permissionResourceWrite)).Delete("/resource-definitions/{id}/fields/{fieldId}", s.deleteField)

		ar.With(requireAdminPermission(permissionAPIClientManage)).Post("/api-clients", s.createAPIClient)
		ar.With(requireAdminPermission(permissionAdminRead)).Get("/api-clients", s.listAPIClients)
		ar.With(requireAdminPermission(permissionAPIClientManage)).Patch("/api-clients/{id}", s.updateAPIClient)
		ar.With(requireAdminPermission(permissionAPIClientManage)).Post("/api-clients/{id}/rotate", s.rotateAPIClient)
		ar.With(requireAdminPermission(permissionAPIClientManage)).Post("/api-clients/{id}/revoke", s.revokeAPIClient)
		ar.With(requireAdminPermission(permissionAPIClientManage)).Delete("/api-clients/{id}", s.deleteAPIClient)
		ar.With(requireAdminPermission(permissionAdminRead)).Get("/api-clients/{id}/usage", s.listAPIClientUsage)
		ar.With(requireAdminPermission(permissionAdminRead)).Get("/api-client-usage", s.listAPIClientUsageOverview)
	})

	if staticHandler != nil {
		r.NotFound(staticHandler.ServeHTTP)
	}
	return r
}

func (s *Server) ConfigureAdminAuth(config adminauth.Config, oidc *adminauth.OIDCManager, sessions *adminauth.SessionManager) {
	s.AdminAuth = &AdminAuthRuntime{Config: config, OIDC: oidc, Sessions: sessions}
}

func (s *Server) SetUsageRecorder(recorder usageRecorder) {
	if recorder != nil {
		s.Usage = recorder
	}
}

func (s *Server) SetRateLimiter(limiter ratelimit.Limiter) {
	if limiter != nil {
		s.Limiter = limiter
	}
}

// NewRouter keeps the simple health-only router for tests/back-compat.
func NewRouter() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(SecurityHeaders)
	r.Use(middleware.Recoverer)
	r.Get("/healthz", Health)
	return r
}

package httpapi

import (
	"net/http"
	"time"

	"github.com/djalmajr/tenancit/server/internal/adminauth"
	"github.com/djalmajr/tenancit/server/internal/auditops"
	"github.com/djalmajr/tenancit/server/internal/crypto"
	"github.com/djalmajr/tenancit/server/internal/ratelimit"
	"github.com/djalmajr/tenancit/server/internal/service"
	appsettings "github.com/djalmajr/tenancit/server/internal/settings"
	"github.com/djalmajr/tenancit/server/internal/store/db"
	"github.com/djalmajr/tenancit/server/internal/telemetry"
	"github.com/djalmajr/tenancit/server/internal/webhook"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Server holds dependencies shared by handlers.
type Server struct {
	AdminTokenHash                    string
	Cryptor                           *crypto.Cryptor
	DB                                *pgxpool.Pool
	Q                                 *db.Queries
	Resolver                          *service.Resolver
	Now                               func() time.Time
	Usage                             usageRecorder
	Limiter                           ratelimit.Limiter
	AdminAuth                         *AdminAuthRuntime
	AdminAuthStore                    *adminauth.PostgresSessionStore
	Settings                          *appsettings.Repository
	Webhooks                          *webhook.TargetRepository
	ReadinessProbes                   []telemetry.Probe
	OperationsReportTokenHash         string
	OperationsReportCredentialVersion string
	TelemetryMiddleware               func(http.Handler) http.Handler
	AuditExports                      *auditops.ExportRepository
}

// NewServer wires a Server from the database pool, cryptor, and admin token.
func NewServer(pool *pgxpool.Pool, c *crypto.Cryptor, adminToken string) *Server {
	q := db.New(pool)
	telemetryMiddleware, _ := telemetry.NewDefaultHTTPMiddleware()
	return &Server{
		AdminTokenHash:      service.HashAPIKey(adminToken),
		Cryptor:             c,
		DB:                  pool,
		Q:                   q,
		Resolver:            service.NewResolver(q, c),
		Now:                 time.Now,
		Usage:               discardUsageRecorder{},
		Limiter:             ratelimit.NewMemory(time.Now),
		Settings:            appsettings.NewRepository(pool, time.Now),
		Webhooks:            webhook.NewTargetRepository(pool, c, nil, nil, false),
		AuditExports:        auditops.NewExportRepository(pool, c, time.Now),
		TelemetryMiddleware: telemetryMiddleware,
	}
}

// Routes builds the full router: health, consumer API (api-key), admin API,
// and the embedded SPA fallback.
func (s *Server) Routes(staticHandler http.Handler) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	if s.TelemetryMiddleware != nil {
		r.Use(s.TelemetryMiddleware)
	}
	r.Use(SecurityHeaders)
	r.Use(middleware.Recoverer)

	r.Get("/healthz", Health)
	r.Get("/readyz", s.Readiness)
	r.With(s.requireOperationsReporter).Post("/v1/operations/reports", s.createOperationalReport)
	r.Get("/v1/auth/config", s.getAdminAuthConfig)

	if s.AdminAuth != nil && s.AdminAuth.Config.Mode == adminauth.ModeOIDC {
		r.Get("/v1/auth/login", s.startOIDCLogin)
		r.Get("/v1/auth/callback", s.completeOIDCLogin)
		r.With(RequireAdminSession(s.AdminAuth.Sessions, s.AdminAuth.Config.CookieName)).Get("/v1/auth/session", s.getAdminSession)
		r.With(
			s.AuditAdminFailures,
			RequireAdminSession(s.AdminAuth.Sessions, s.AdminAuth.Config.CookieName),
			RequireAdminCSRF(s.AdminAuth.Config.AdminOrigin),
		).Post("/v1/auth/logout", s.logoutAdminSession)
	}

	r.Group(func(cr chi.Router) {
		cr.Use(RequireAPIKey(s.Q, s.Now))
		cr.With(RequireAPIClientScope(service.ScopeTenantIdentify), EnforceAPIClientRateLimit(s.Limiter, s.Usage, "identify", s.Now), RecordAPIUsage(s.Usage, "identify", s.Now)).Get("/v1/identify", s.handleIdentify)
		cr.With(RequireAPIClientScope(service.ScopeResourceResolve), EnforceAPIClientRateLimit(s.Limiter, s.Usage, "resolve", s.Now), RecordAPIUsage(s.Usage, "resolve", s.Now)).Get("/v1/resolve", s.handleResolve)
		cr.With(RequireAPIClientScope(service.ScopeResourceResolve), EnforceAPIClientRateLimit(s.Limiter, s.Usage, "resolve", s.Now), RecordAPIUsage(s.Usage, "resolve", s.Now)).Get("/v1/resolve/{hostname}/resources/{definitionKey}", s.handleResolveOne)
		cr.With(RequireAPIClientScope(service.ScopeEventsRead), EnforceAPIClientRateLimit(s.Limiter, s.Usage, "events", s.Now), RecordAPIUsage(s.Usage, "events", s.Now)).Get("/v1/events", s.listChangeFeed)
	})

	r.Route("/v1/admin", func(ar chi.Router) {
		ar.Use(s.AuditAdminFailures)
		if s.AdminAuth != nil && s.AdminAuth.Config.Mode == adminauth.ModeOIDC {
			breakGlassHash := ""
			breakGlassVersion := ""
			if s.AdminAuth.Config.BreakGlass.Enabled {
				breakGlassHash = s.AdminAuth.Config.BreakGlass.TokenHash
				breakGlassVersion = s.AdminAuth.Config.BreakGlass.Version
			}
			ar.Use(RequireAdminIdentity(s.AdminAuth.Sessions, s.AdminAuth.Config.CookieName, breakGlassHash, breakGlassVersion))
			ar.Use(RequireAdminCSRF(s.AdminAuth.Config.AdminOrigin))
		} else {
			ar.Use(RequireAdminToken(s.AdminTokenHash))
		}

		ar.With(requireAdminPermission(permissionAdminRead)).Get("/overview", s.overview)
		ar.With(requireAdminPermission(permissionAdminRead)).Get("/operational-health", s.operationalHealth)
		ar.With(requireAdminPermission(permissionAuditRead)).Get("/audit-events", s.listAuditEvents)
		ar.With(requireAdminPermission(permissionAuditRead)).Get("/audit-health", s.auditHealth)
		ar.With(requireAdminPermission(permissionAuditManage)).Get("/audit-legal-holds", s.listAuditLegalHolds)
		ar.With(requireAdminPermission(permissionAuditManage)).Post("/audit-legal-holds", s.createAuditLegalHold)
		ar.With(requireAdminPermission(permissionAuditManage)).Post("/audit-legal-holds/{id}/release", s.releaseAuditLegalHold)
		ar.With(requireAdminPermission(permissionAuditExport)).Post("/audit-exports", s.createAuditExport)
		ar.With(requireAdminPermission(permissionAuditExport)).Get("/audit-exports/{id}", s.getAuditExport)
		ar.With(requireAdminPermission(permissionAuditExport)).Get("/audit-exports/{id}/download", s.downloadAuditExport)

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

		ar.With(requireAdminPermission(permissionAdminRead)).Get("/settings", s.getSettings)
		ar.With(requireAdminPermission(permissionSettingsManage)).Patch("/settings", s.updateSettings)
		ar.With(requireAdminPermission(permissionSessionManage)).Get("/sessions", s.listAdminSessions)
		ar.With(requireAdminPermission(permissionSessionManage)).Delete("/sessions/{id}", s.revokeAdminSession)
		ar.With(requireAdminPermission(permissionSessionManage)).Post("/sessions/revoke-principal", s.revokeAdminPrincipalSessions)
		ar.With(requireAdminPermission(permissionIntegrationManage)).Get("/webhook-targets", s.listWebhookTargets)
		ar.With(requireAdminPermission(permissionIntegrationManage)).Post("/webhook-targets", s.createWebhookTarget)
		ar.With(requireAdminPermission(permissionIntegrationManage)).Put("/webhook-targets/{id}/status", s.setWebhookTargetStatus)
		ar.With(requireAdminPermission(permissionIntegrationManage)).Get("/webhook-deliveries", s.listWebhookDeliveries)
		ar.With(requireAdminPermission(permissionIntegrationManage)).Post("/webhook-deliveries/{id}/replay", s.replayWebhookDelivery)
		ar.With(requireAdminPermission(permissionIntegrationManage)).Get("/webhook-overview", s.webhookOverview)
	})

	if staticHandler != nil {
		r.NotFound(staticHandler.ServeHTTP)
	}
	return r
}

func (s *Server) SetAdminAuthStore(store *adminauth.PostgresSessionStore) {
	s.AdminAuthStore = store
}

func (s *Server) SetWebhookTargets(repository *webhook.TargetRepository) {
	if repository != nil {
		s.Webhooks = repository
	}
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

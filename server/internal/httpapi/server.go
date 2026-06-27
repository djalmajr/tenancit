package httpapi

import (
	"net/http"

	"github.com/djalmajr/tenancit/server/internal/crypto"
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
	}
}

// Routes builds the full router: health, consumer API (api-key), admin API,
// and the embedded SPA fallback.
func (s *Server) Routes(staticHandler http.Handler) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)

	r.Get("/healthz", Health)

	r.Group(func(cr chi.Router) {
		cr.Use(RequireAPIKey(s.Q))
		cr.Get("/v1/identify", s.handleIdentify)
		cr.Get("/v1/resolve", s.handleResolve)
		cr.Get("/v1/resolve/{hostname}/resources/{definitionKey}", s.handleResolveOne)
	})

	r.Route("/v1/admin", func(ar chi.Router) {
		ar.Use(RequireAdminToken(s.AdminTokenHash))

		ar.Get("/overview", s.overview)

		ar.Post("/tenants", s.createTenant)
		ar.Get("/tenants", s.listTenants)
		ar.Get("/tenants/{id}", s.getTenant)
		ar.Put("/tenants/{id}", s.updateTenant)
		ar.Delete("/tenants/{id}", s.deleteTenant)
		ar.Post("/tenants/{id}/domains", s.addDomain)
		ar.Get("/tenants/{id}/domains", s.listTenantDomains)
		ar.Delete("/tenants/{id}/domains/{domainId}", s.deleteDomain)
		ar.Post("/tenants/{id}/resources", s.createResource)
		ar.Get("/tenants/{id}/resources", s.listTenantResources)
		ar.Put("/tenants/{id}/resources/{resourceId}/status", s.setResourceStatus)
		ar.Delete("/tenants/{id}/resources/{resourceId}", s.deleteResource)

		ar.Post("/resource-definitions", s.createDefinition)
		ar.Get("/resource-definitions", s.listDefinitions)
		ar.Get("/resource-definitions/{id}", s.getDefinition)
		ar.Put("/resource-definitions/{id}/status", s.setDefinitionStatus)
		ar.Post("/resource-definitions/{id}/fields", s.addField)
		ar.Delete("/resource-definitions/{id}/fields/{fieldId}", s.deleteField)

		ar.Post("/api-clients", s.createAPIClient)
		ar.Get("/api-clients", s.listAPIClients)
		ar.Put("/api-clients/{id}/status", s.setAPIClientStatus)
	})

	if staticHandler != nil {
		r.NotFound(staticHandler.ServeHTTP)
	}
	return r
}

// NewRouter keeps the simple health-only router for tests/back-compat.
func NewRouter() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)
	r.Get("/healthz", Health)
	return r
}

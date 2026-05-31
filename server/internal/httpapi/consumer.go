package httpapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (s *Server) handleResolve(w http.ResponseWriter, r *http.Request) {
	hostname := r.URL.Query().Get("hostname")
	if hostname == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "hostname required"})
		return
	}
	res, err := s.Resolver.ByHostname(r.Context(), hostname)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "tenant not found"})
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (s *Server) handleResolveOne(w http.ResponseWriter, r *http.Request) {
	hostname := chi.URLParam(r, "hostname")
	defKey := chi.URLParam(r, "definitionKey")
	res, found, err := s.Resolver.ByHostnameAndDefinition(r.Context(), hostname, defKey)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "tenant not found"})
		return
	}
	if !found {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "resource not found"})
		return
	}
	writeJSON(w, http.StatusOK, res)
}

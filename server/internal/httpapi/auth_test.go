package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/djalmajr/tenancit/server/internal/service"
	"github.com/djalmajr/tenancit/server/internal/store/db"
)

type fakeLookup struct{ clients map[string]db.ApiClient }

func (f fakeLookup) GetAPIClientByHash(_ context.Context, h string) (db.ApiClient, error) {
	c, ok := f.clients[h]
	if !ok {
		return db.ApiClient{}, errors.New("not found")
	}
	return c, nil
}

func protected(lk apiKeyLookup) http.Handler {
	ok := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) })
	return RequireAPIKey(lk)(ok)
}

func TestAuth_NoToken_401(t *testing.T) {
	rec := httptest.NewRecorder()
	protected(fakeLookup{}).ServeHTTP(rec, httptest.NewRequest("GET", "/x", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("code=%d want 401", rec.Code)
	}
}

func TestAuth_ValidToken_200(t *testing.T) {
	tok := "tnc_abc"
	lk := fakeLookup{clients: map[string]db.ApiClient{service.HashAPIKey(tok): {Status: "active"}}}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/x", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	protected(lk).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("code=%d want 200", rec.Code)
	}
}

func TestAuth_RevokedToken_401(t *testing.T) {
	tok := "tnc_revoked"
	lk := fakeLookup{clients: map[string]db.ApiClient{service.HashAPIKey(tok): {Status: "revoked"}}}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/x", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	protected(lk).ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("code=%d want 401 for revoked", rec.Code)
	}
}

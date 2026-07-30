package spa

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestHandlerServesConfiguredBasePathAndRejectsOtherRoots(t *testing.T) {
	handler, err := Handler("/tenancit")
	if err != nil {
		t.Fatal(err)
	}

	t.Run("canonical redirect", func(t *testing.T) {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/tenancit", nil))
		if rec.Code != http.StatusPermanentRedirect || rec.Header().Get("Location") != "/tenancit/" {
			t.Fatalf("redirect = %d %q", rec.Code, rec.Header().Get("Location"))
		}
	})

	t.Run("deep link", func(t *testing.T) {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/tenancit/tenants/example", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, body=%s", rec.Code, rec.Body)
		}
		body := rec.Body.String()
		for _, want := range []string{
			`<base href="/tenancit/">`,
			`<meta name="tenancit-base-path" content="/tenancit">`,
		} {
			if !strings.Contains(body, want) {
				t.Fatalf("index missing %q: %s", want, body)
			}
		}
		if strings.Count(body, `name="tenancit-base-path"`) != 1 ||
			strings.Count(body, `<base href=`) != 1 {
			t.Fatalf("runtime base-path markers were duplicated: %s", body)
		}
	})

	for _, requestPath := range []string{"/", "/tenants", "/tenancit-other", "/tenancit/v1/auth/config"} {
		requestPath := requestPath
		t.Run("not found "+requestPath, func(t *testing.T) {
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, requestPath, nil))
			if rec.Code != http.StatusNotFound {
				t.Fatalf("%s status = %d, want 404", requestPath, rec.Code)
			}
		})
	}
}

func TestHandlerPreservesRootModeForBackwardCompatibility(t *testing.T) {
	handler, err := Handler("/")
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/tenants/example", nil))
	if rec.Code != http.StatusOK ||
		!strings.Contains(rec.Body.String(), `<base href="/">`) ||
		!strings.Contains(rec.Body.String(), `content="/"`) {
		t.Fatalf("root SPA response = %d %s", rec.Code, rec.Body)
	}
}

func TestHandlerRejectsInvalidBasePaths(t *testing.T) {
	// Mutation captured: reducing base-path validation to a leading-slash check
	// lets authority-like redirect targets reach http.Redirect.
	for _, value := range []string{
		"tenancit",
		"//tenancit",
		`/\tenancit`,
		"/%2f%2fattacker.example",
		"/%5cattacker.example",
		"/tenancit//admin",
		"/tenancit/../admin",
		"/tenancit/%2e%2e/admin",
		`/tenancit\admin`,
		"/tenancit?debug=true",
		"/tenancit#fragment",
	} {
		if _, err := Handler(value); err == nil {
			t.Fatalf("Handler(%q) accepted an invalid base path", value)
		}
	}
}

func TestHandlerCanonicalRedirectRemainsOriginRelative(t *testing.T) {
	// Mutation captured: returning a protocol-relative Location gives the
	// browser an external authority instead of the configured internal path.
	for _, value := range []string{"/tenancit", "/platform/tenancit", "/tenant-admin_2"} {
		handler, err := Handler(value)
		if err != nil {
			t.Fatalf("Handler(%q): %v", value, err)
		}

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, value, nil))
		location := rec.Header().Get("Location")
		parsed, parseErr := url.Parse(location)
		if parseErr != nil {
			t.Fatalf("redirect %q is not a URL: %v", location, parseErr)
		}
		if rec.Code != http.StatusPermanentRedirect ||
			location != value+"/" ||
			parsed.IsAbs() ||
			parsed.Host != "" ||
			strings.HasPrefix(location, "//") ||
			strings.HasPrefix(location, `/\`) {
			t.Fatalf("Handler(%q) redirect = %d %q", value, rec.Code, location)
		}
	}
}

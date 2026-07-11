package adminauth

import (
	"strings"
	"testing"
)

func TestLoadConfigRejectsMissingOrPartialMode(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
		want string
	}{
		{name: "missing mode", env: map[string]string{}, want: "TENANCIT_ADMIN_AUTH_MODE"},
		{name: "unknown mode", env: map[string]string{"TENANCIT_ADMIN_AUTH_MODE": "magic"}, want: "unsupported"},
		{
			name: "partial oidc",
			env: map[string]string{
				"TENANCIT_ADMIN_AUTH_MODE": "oidc",
				"TENANCIT_ADMIN_ORIGIN":    "https://tenancit.example.test",
				"TENANCIT_OIDC_ISSUER":     "https://id.example.test",
			},
			want: "TENANCIT_OIDC_CLIENT_ID",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := LoadConfig(mapGetter(tt.env))
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error=%v want containing %q", err, tt.want)
			}
		})
	}
}

func TestLoadConfigAcceptsCompleteOIDCAndDerivesCallback(t *testing.T) {
	cfg, err := LoadConfig(mapGetter(map[string]string{
		"TENANCIT_ADMIN_AUTH_MODE":    "oidc",
		"TENANCIT_ADMIN_ORIGIN":       "https://tenancit.example.test",
		"TENANCIT_OIDC_ISSUER":        "https://id.example.test/realms/platform",
		"TENANCIT_OIDC_CLIENT_ID":     "tenancit",
		"TENANCIT_OIDC_CLIENT_SECRET": "client-secret",
		"TENANCIT_OIDC_ROLE_CLAIM":    "groups",
		"TENANCIT_OIDC_ROLE_MAPPINGS": `{"/tenancit/viewers":"viewer","/tenancit/operators":"operator"}`,
	}))
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.Mode != ModeOIDC || cfg.OIDC.RedirectURL != "https://tenancit.example.test/v1/auth/callback" {
		t.Fatalf("config=%+v", cfg)
	}
	if cfg.CookieName != "__Host-tenancit_session" || !cfg.CookieSecure {
		t.Fatalf("cookie=%q secure=%v", cfg.CookieName, cfg.CookieSecure)
	}
	if cfg.OIDC.RoleMappings["/tenancit/operators"] != RoleOperator {
		t.Fatalf("role mappings=%v", cfg.OIDC.RoleMappings)
	}
}

func TestLoadConfigAllowsLoopbackOnlyInExplicitDevMode(t *testing.T) {
	base := map[string]string{
		"TENANCIT_ADMIN_AUTH_MODE":    "oidc",
		"TENANCIT_ADMIN_ORIGIN":       "http://localhost:5180",
		"TENANCIT_OIDC_ISSUER":        "http://localhost:5556/dex",
		"TENANCIT_OIDC_CLIENT_ID":     "tenancit",
		"TENANCIT_OIDC_CLIENT_SECRET": "client-secret",
		"TENANCIT_OIDC_ROLE_CLAIM":    "groups",
		"TENANCIT_OIDC_ROLE_MAPPINGS": `{"admins":"security_admin"}`,
	}
	if _, err := LoadConfig(mapGetter(base)); err == nil {
		t.Fatal("insecure origin accepted outside explicit dev mode")
	}
	base["TENANCIT_DEV_MODE"] = "true"
	cfg, err := LoadConfig(mapGetter(base))
	if err != nil {
		t.Fatalf("LoadConfig dev: %v", err)
	}
	if cfg.CookieSecure || cfg.CookieName == "__Host-tenancit_session" {
		t.Fatalf("dev cookie=%q secure=%v", cfg.CookieName, cfg.CookieSecure)
	}
}

func TestLoadConfigRestrictsLegacySharedTokenToDev(t *testing.T) {
	env := map[string]string{
		"TENANCIT_ADMIN_AUTH_MODE": "legacy_shared_token",
		"TENANCIT_ADMIN_TOKEN":     "strong-development-token",
	}
	if _, err := LoadConfig(mapGetter(env)); err == nil {
		t.Fatal("legacy shared token accepted outside dev mode")
	}
	env["TENANCIT_DEV_MODE"] = "true"
	if _, err := LoadConfig(mapGetter(env)); err != nil {
		t.Fatalf("LoadConfig dev legacy mode: %v", err)
	}
}

func mapGetter(values map[string]string) func(string) string {
	return func(key string) string { return values[key] }
}

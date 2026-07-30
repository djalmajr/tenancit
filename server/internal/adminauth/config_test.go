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
	if cfg.BasePath != "/" || cfg.OIDC.BasePath != "/" ||
		cfg.CookieName != "__Host-tenancit_session" || cfg.CookiePath != "/" || !cfg.CookieSecure {
		t.Fatalf("base=%q oidc_base=%q cookie=%q path=%q secure=%v", cfg.BasePath, cfg.OIDC.BasePath, cfg.CookieName, cfg.CookiePath, cfg.CookieSecure)
	}
	if cfg.OIDC.RoleMappings["/tenancit/operators"] != RoleOperator {
		t.Fatalf("role mappings=%v", cfg.OIDC.RoleMappings)
	}
}

func TestLoadConfigDerivesPrefixedOIDCCallbackAndCookieScope(t *testing.T) {
	// Mutation captured by the initial RED: ignoring TENANCIT_BASE_PATH kept
	// callback and cookie scope at the shared host root.
	cfg, err := LoadConfig(mapGetter(map[string]string{
		"TENANCIT_ADMIN_AUTH_MODE":    "oidc",
		"TENANCIT_ADMIN_ORIGIN":       "https://admin.example.test",
		"TENANCIT_BASE_PATH":          "/tenancit",
		"TENANCIT_OIDC_ISSUER":        "https://id.example.test/realms/platform",
		"TENANCIT_OIDC_CLIENT_ID":     "tenancit",
		"TENANCIT_OIDC_CLIENT_SECRET": "client-secret",
		"TENANCIT_OIDC_ROLE_CLAIM":    "groups",
		"TENANCIT_OIDC_ROLE_MAPPINGS": `{"admins":"security_admin"}`,
	}))
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.BasePath != "/tenancit" || cfg.OIDC.BasePath != "/tenancit" {
		t.Fatalf("base path config=%+v oidc=%+v", cfg, cfg.OIDC)
	}
	if cfg.OIDC.RedirectURL != "https://admin.example.test/tenancit/v1/auth/callback" {
		t.Fatalf("redirect URL=%q", cfg.OIDC.RedirectURL)
	}
	if cfg.CookieName != "__Secure-tenancit_session" || cfg.CookiePath != "/tenancit" || !cfg.CookieSecure {
		t.Fatalf("cookie=%q path=%q secure=%v", cfg.CookieName, cfg.CookiePath, cfg.CookieSecure)
	}
}

func TestLoadConfigDefaultsBasePathToRootAndRejectsInvalidValues(t *testing.T) {
	base := map[string]string{
		"TENANCIT_ADMIN_AUTH_MODE": "legacy_shared_token",
		"TENANCIT_ADMIN_TOKEN":     "strong-development-token",
		"TENANCIT_DEV_MODE":        "true",
	}
	cfg, err := LoadConfig(mapGetter(base))
	if err != nil {
		t.Fatalf("LoadConfig root: %v", err)
	}
	if cfg.BasePath != "/" || cfg.CookiePath != "/" {
		t.Fatalf("root base/cookie path=%q/%q", cfg.BasePath, cfg.CookiePath)
	}

	// Mutation captured by the initial RED: accepting a malformed prefix can
	// mount administrative routes outside their intended URL boundary.
	for _, value := range []string{
		"tenancit",
		"//tenancit",
		"/tenancit//admin",
		"/tenancit/../admin",
		`/tenancit\admin`,
		"/tenancit?debug=true",
		"/tenancit#fragment",
	} {
		base["TENANCIT_BASE_PATH"] = value
		if _, err := LoadConfig(mapGetter(base)); err == nil || !strings.Contains(err.Error(), "TENANCIT_BASE_PATH") {
			t.Fatalf("base path %q error=%v", value, err)
		}
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

func TestLoadConfigRequiresExplicitStrongBreakGlass(t *testing.T) {
	base := map[string]string{
		"TENANCIT_ADMIN_AUTH_MODE":    "oidc",
		"TENANCIT_ADMIN_ORIGIN":       "https://tenancit.example.test",
		"TENANCIT_OIDC_ISSUER":        "https://id.example.test",
		"TENANCIT_OIDC_CLIENT_ID":     "tenancit",
		"TENANCIT_OIDC_CLIENT_SECRET": "client-secret",
		"TENANCIT_OIDC_ROLE_CLAIM":    "groups",
		"TENANCIT_OIDC_ROLE_MAPPINGS": `{"admins":"security_admin"}`,
	}

	base["TENANCIT_ADMIN_TOKEN"] = strings.Repeat("x", 32)
	if _, err := LoadConfig(mapGetter(base)); err == nil || !strings.Contains(err.Error(), "requires TENANCIT_BREAK_GLASS_ENABLED") {
		t.Fatalf("disabled break-glass error=%v", err)
	}
	base["TENANCIT_BREAK_GLASS_ENABLED"] = "true"
	if _, err := LoadConfig(mapGetter(base)); err == nil || !strings.Contains(err.Error(), "TENANCIT_BREAK_GLASS_VERSION") {
		t.Fatalf("missing version error=%v", err)
	}
	base["TENANCIT_BREAK_GLASS_VERSION"] = "rotation-2026-07"
	cfg, err := LoadConfig(mapGetter(base))
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if !cfg.BreakGlass.Enabled || cfg.BreakGlass.Version != "rotation-2026-07" {
		t.Fatalf("break-glass=%+v", cfg.BreakGlass)
	}
	if cfg.BreakGlass.TokenHash == base["TENANCIT_ADMIN_TOKEN"] || cfg.BreakGlass.TokenHash != HashCredential(base["TENANCIT_ADMIN_TOKEN"]) {
		t.Fatal("break-glass configuration retained a raw credential")
	}
}

func mapGetter(values map[string]string) func(string) string {
	return func(key string) string { return values[key] }
}

package adminauth

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"
)

type Mode string

const (
	ModeOIDC              Mode = "oidc"
	ModeLegacySharedToken Mode = "legacy_shared_token"
)

type Role string

const (
	RoleViewer        Role = "viewer"
	RoleOperator      Role = "operator"
	RoleSecurityAdmin Role = "security_admin"
)

type OIDCConfig struct {
	Issuer       string
	ClientID     string
	ClientSecret string
	RedirectURL  string
	RoleClaim    string
	RoleMappings map[string]Role
}

type Config struct {
	Mode            Mode
	DevMode         bool
	AdminOrigin     string
	CookieName      string
	CookieSecure    bool
	SessionAbsolute time.Duration
	SessionIdle     time.Duration
	LegacyToken     string
	BreakGlass      BreakGlassConfig
	OIDC            OIDCConfig
}

type BreakGlassConfig struct {
	Enabled   bool
	TokenHash string
	Version   string
}

// LoadConfig validates the complete administrative authentication boundary.
// It deliberately accepts a getter so tests and callers do not need to mutate
// process-global environment state.
func LoadConfig(getenv func(string) string) (Config, error) {
	if getenv == nil {
		return Config{}, errors.New("environment getter is required")
	}
	mode := Mode(strings.TrimSpace(getenv("TENANCIT_ADMIN_AUTH_MODE")))
	if mode == "" {
		return Config{}, errors.New("TENANCIT_ADMIN_AUTH_MODE is required")
	}
	devMode := strings.EqualFold(strings.TrimSpace(getenv("TENANCIT_DEV_MODE")), "true")
	cfg := Config{
		Mode:            mode,
		DevMode:         devMode,
		SessionAbsolute: 8 * time.Hour,
		SessionIdle:     30 * time.Minute,
	}

	switch mode {
	case ModeLegacySharedToken:
		if !devMode {
			return Config{}, errors.New("legacy_shared_token requires TENANCIT_DEV_MODE=true")
		}
		cfg.LegacyToken = strings.TrimSpace(getenv("TENANCIT_ADMIN_TOKEN"))
		if len(cfg.LegacyToken) < 16 {
			return Config{}, errors.New("TENANCIT_ADMIN_TOKEN must contain at least 16 characters in dev mode")
		}
		cfg.CookieName = "tenancit_session"
		return cfg, nil
	case ModeOIDC:
		return loadOIDCConfig(cfg, getenv)
	default:
		return Config{}, fmt.Errorf("unsupported TENANCIT_ADMIN_AUTH_MODE %q", mode)
	}
}

func loadOIDCConfig(cfg Config, getenv func(string) string) (Config, error) {
	breakGlassEnabled, err := parseExplicitBool("TENANCIT_BREAK_GLASS_ENABLED", getenv("TENANCIT_BREAK_GLASS_ENABLED"))
	if err != nil {
		return Config{}, err
	}
	breakGlassToken := strings.TrimSpace(getenv("TENANCIT_ADMIN_TOKEN"))
	breakGlassVersion := strings.TrimSpace(getenv("TENANCIT_BREAK_GLASS_VERSION"))
	if !breakGlassEnabled && (breakGlassToken != "" || breakGlassVersion != "") {
		return Config{}, errors.New("break-glass token or version requires TENANCIT_BREAK_GLASS_ENABLED=true")
	}
	if breakGlassEnabled {
		if len(breakGlassToken) < 32 {
			return Config{}, errors.New("TENANCIT_ADMIN_TOKEN must contain at least 32 characters for break-glass")
		}
		if breakGlassVersion == "" {
			return Config{}, errors.New("TENANCIT_BREAK_GLASS_VERSION is required when break-glass is enabled")
		}
		cfg.BreakGlass = BreakGlassConfig{Enabled: true, TokenHash: HashCredential(breakGlassToken), Version: breakGlassVersion}
	}
	origin, err := validateEndpoint("TENANCIT_ADMIN_ORIGIN", getenv("TENANCIT_ADMIN_ORIGIN"), cfg.DevMode, true)
	if err != nil {
		return Config{}, err
	}
	issuer, err := validateEndpoint("TENANCIT_OIDC_ISSUER", getenv("TENANCIT_OIDC_ISSUER"), cfg.DevMode, false)
	if err != nil {
		return Config{}, err
	}
	clientID := strings.TrimSpace(getenv("TENANCIT_OIDC_CLIENT_ID"))
	if clientID == "" {
		return Config{}, errors.New("TENANCIT_OIDC_CLIENT_ID is required")
	}
	clientSecret := strings.TrimSpace(getenv("TENANCIT_OIDC_CLIENT_SECRET"))
	if clientSecret == "" {
		return Config{}, errors.New("TENANCIT_OIDC_CLIENT_SECRET is required")
	}
	roleClaim := strings.TrimSpace(getenv("TENANCIT_OIDC_ROLE_CLAIM"))
	if roleClaim == "" {
		return Config{}, errors.New("TENANCIT_OIDC_ROLE_CLAIM is required")
	}
	roleMappings, err := parseRoleMappings(getenv("TENANCIT_OIDC_ROLE_MAPPINGS"))
	if err != nil {
		return Config{}, err
	}

	cfg.AdminOrigin = strings.TrimSuffix(origin.String(), "/")
	cfg.CookieSecure = origin.Scheme == "https"
	cfg.CookieName = "__Host-tenancit_session"
	if !cfg.CookieSecure {
		cfg.CookieName = "tenancit_session"
	}
	cfg.OIDC = OIDCConfig{
		Issuer:       strings.TrimSuffix(issuer.String(), "/"),
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  cfg.AdminOrigin + "/v1/auth/callback",
		RoleClaim:    roleClaim,
		RoleMappings: roleMappings,
	}
	return cfg, nil
}

func parseExplicitBool(name, raw string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "false":
		return false, nil
	case "true":
		return true, nil
	default:
		return false, fmt.Errorf("%s must be true or false", name)
	}
}

func validateEndpoint(name, raw string, devMode, originOnly bool) (*url.URL, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("%s is required", name)
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return nil, fmt.Errorf("%s must be an absolute URL without credentials, query, or fragment", name)
	}
	if originOnly && u.Path != "" && u.Path != "/" {
		return nil, fmt.Errorf("%s must be an origin without a path", name)
	}
	if u.Scheme == "https" {
		return u, nil
	}
	host := u.Hostname()
	if !devMode || u.Scheme != "http" || !(host == "localhost" || net.ParseIP(host).IsLoopback()) {
		return nil, fmt.Errorf("%s must use https outside explicit loopback dev mode", name)
	}
	return u, nil
}

func parseRoleMappings(raw string) (map[string]Role, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, errors.New("TENANCIT_OIDC_ROLE_MAPPINGS is required")
	}
	var mappings map[string]Role
	if err := json.Unmarshal([]byte(raw), &mappings); err != nil {
		return nil, fmt.Errorf("TENANCIT_OIDC_ROLE_MAPPINGS must be a JSON object: %w", err)
	}
	if len(mappings) == 0 {
		return nil, errors.New("TENANCIT_OIDC_ROLE_MAPPINGS must contain at least one mapping")
	}
	for group, role := range mappings {
		if strings.TrimSpace(group) == "" {
			return nil, errors.New("TENANCIT_OIDC_ROLE_MAPPINGS contains an empty claim value")
		}
		switch role {
		case RoleViewer, RoleOperator, RoleSecurityAdmin:
		default:
			return nil, fmt.Errorf("TENANCIT_OIDC_ROLE_MAPPINGS contains unsupported role %q", role)
		}
	}
	return mappings, nil
}

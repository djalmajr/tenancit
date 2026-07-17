package service

import (
	"errors"
	"time"
)

const (
	ScopeTenantIdentify  = "tenant:identify"
	ScopeResourceResolve = "resource:resolve"
	ScopeEventsRead      = "events:read"
	// ScopeTenantList lets a consumer enumerate tenant identities (slug/name/
	// status only — never resources or secrets). Useful for control-plane
	// bridges that mirror the tenant directory (e.g. platform shells).
	ScopeTenantList = "tenant:list"
	// ScopeResourceWrite lets a consumer update values of NON-SECRET fields on
	// existing resources (identified by tenant slug + alias). Secret fields are
	// always rejected — secret management stays in the admin console.
	ScopeResourceWrite = "resource:write"
)

var (
	ErrInvalidScope      = errors.New("invalid_scope")
	ErrInvalidRPM        = errors.New("invalid_rpm")
	ErrInvalidExpiration = errors.New("invalid_expiration")
)

func ValidateAPIClientPolicy(now time.Time, scopes []string, rpm int32, expiresAt time.Time) error {
	if len(scopes) == 0 {
		return ErrInvalidScope
	}
	seen := make(map[string]struct{}, len(scopes))
	for _, scope := range scopes {
		if scope != ScopeTenantIdentify && scope != ScopeResourceResolve && scope != ScopeEventsRead && scope != ScopeTenantList && scope != ScopeResourceWrite {
			return ErrInvalidScope
		}
		if _, duplicate := seen[scope]; duplicate {
			return ErrInvalidScope
		}
		seen[scope] = struct{}{}
	}
	if rpm <= 0 {
		return ErrInvalidRPM
	}
	if !expiresAt.After(now) || expiresAt.After(now.Add(365*24*time.Hour)) {
		return ErrInvalidExpiration
	}
	return nil
}

func EffectiveAPIClientStatus(status string, expiresAt time.Time, now time.Time) string {
	if status == "revoked" {
		return "revoked"
	}
	if !expiresAt.IsZero() && !now.Before(expiresAt) {
		return "expired"
	}
	return "active"
}

func APITokenPreview(token string) string {
	const visible = 12
	if len(token) <= visible {
		return token + "…"
	}
	return token[:visible] + "…"
}

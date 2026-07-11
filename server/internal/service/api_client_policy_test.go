package service

import (
	"testing"
	"time"
)

func TestValidateAPIClientPolicyRejectsInvalidInputs(t *testing.T) {
	now := time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name   string
		scopes []string
		rpm    int32
		expiry time.Time
		want   error
	}{
		{name: "empty scopes", rpm: 300, expiry: now.Add(90 * 24 * time.Hour), want: ErrInvalidScope},
		{name: "unknown scope", scopes: []string{"admin"}, rpm: 300, expiry: now.Add(90 * 24 * time.Hour), want: ErrInvalidScope},
		{name: "zero rpm", scopes: []string{ScopeTenantIdentify}, expiry: now.Add(90 * 24 * time.Hour), want: ErrInvalidRPM},
		{name: "past expiry", scopes: []string{ScopeTenantIdentify}, rpm: 300, expiry: now, want: ErrInvalidExpiration},
		{name: "over one year", scopes: []string{ScopeTenantIdentify}, rpm: 300, expiry: now.Add(366 * 24 * time.Hour), want: ErrInvalidExpiration},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidateAPIClientPolicy(now, tt.scopes, tt.rpm, tt.expiry); err != tt.want {
				t.Fatalf("ValidateAPIClientPolicy() = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestValidateAPIClientPolicyAcceptsClosedScopes(t *testing.T) {
	now := time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC)
	err := ValidateAPIClientPolicy(now,
		[]string{ScopeTenantIdentify, ScopeResourceResolve}, 300, now.Add(90*24*time.Hour))
	if err != nil {
		t.Fatalf("ValidateAPIClientPolicy() = %v", err)
	}
}

func TestEffectiveAPIClientStatus(t *testing.T) {
	now := time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC)
	if got := EffectiveAPIClientStatus("revoked", now.Add(time.Hour), now); got != "revoked" {
		t.Fatalf("revoked status = %q", got)
	}
	if got := EffectiveAPIClientStatus("active", now, now); got != "expired" {
		t.Fatalf("expiry boundary status = %q", got)
	}
	if got := EffectiveAPIClientStatus("active", now.Add(time.Hour), now); got != "active" {
		t.Fatalf("active status = %q", got)
	}
}

func TestAPITokenPreviewDoesNotExposeWholeToken(t *testing.T) {
	token := "tnc_fixture_value_long"
	preview := APITokenPreview(token)
	if preview == token || preview != "tnc_01234567…" {
		t.Fatalf("APITokenPreview() = %q", preview)
	}
}

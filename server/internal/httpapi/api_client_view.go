package httpapi

import (
	"time"

	"github.com/djalmajr/tenancit/server/internal/service"
	"github.com/djalmajr/tenancit/server/internal/store/db"
	"github.com/google/uuid"
)

type apiClientView struct {
	ID              uuid.UUID  `json:"id"`
	Name            string     `json:"name"`
	KeyPreview      *string    `json:"key_preview,omitempty"`
	Scopes          []string   `json:"scopes"`
	RPMLimit        *int32     `json:"rpm_limit,omitempty"`
	ExpiresAt       *time.Time `json:"expires_at,omitempty"`
	LastUsedAt      *time.Time `json:"last_used_at,omitempty"`
	RevokedAt       *time.Time `json:"revoked_at,omitempty"`
	Status          string     `json:"status"`
	LegacyUnbounded bool       `json:"legacy_unbounded"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type createAPIClientResponse struct {
	Client apiClientView `json:"client"`
	Token  string        `json:"token"`
}

func newAPIClientView(client db.ApiClient, scopes []string, now time.Time) apiClientView {
	var expiresAt, lastUsedAt, revokedAt *time.Time
	if client.ExpiresAt.Valid {
		expiresAt = &client.ExpiresAt.Time
	}
	if client.LastUsedAt.Valid {
		lastUsedAt = &client.LastUsedAt.Time
	}
	if client.RevokedAt.Valid {
		revokedAt = &client.RevokedAt.Time
	}
	effectiveStatus := client.Status
	if expiresAt != nil {
		effectiveStatus = service.EffectiveAPIClientStatus(client.Status, *expiresAt, now)
	}
	return apiClientView{
		ID: client.ID, Name: client.Name, KeyPreview: client.TokenPreview,
		Scopes: scopes, RPMLimit: client.RpmLimit, ExpiresAt: expiresAt,
		LastUsedAt: lastUsedAt, RevokedAt: revokedAt, Status: effectiveStatus,
		LegacyUnbounded: client.RpmLimit == nil || !client.ExpiresAt.Valid,
		CreatedAt:       client.CreatedAt, UpdatedAt: client.UpdatedAt,
	}
}

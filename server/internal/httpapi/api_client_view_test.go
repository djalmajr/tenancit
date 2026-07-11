package httpapi

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/djalmajr/tenancit/server/internal/store/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func TestAPIClientViewPublishesFinalGovernanceContract(t *testing.T) {
	now := time.Date(2026, 7, 11, 21, 0, 0, 0, time.UTC)
	preview := "tnc_abcd1234"
	rpm := int32(300)
	view := newAPIClientView(db.ApiClient{
		ID: uuid.New(), Name: "service", Status: "active", CreatedAt: now,
		TokenPreview: &preview, RpmLimit: &rpm,
		ExpiresAt: pgtype.Timestamptz{Time: now.Add(90 * 24 * time.Hour), Valid: true},
		UpdatedAt: now,
	}, []string{"tenant:identify"}, now)
	payload, err := json.Marshal(view)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"key_preview", "rpm_limit", "expires_at"} {
		if _, ok := decoded[key]; !ok {
			t.Fatalf("final contract omitted %q: %s", key, payload)
		}
	}
	if _, ok := decoded["legacy_unbounded"]; ok {
		t.Fatalf("transient legacy marker leaked into final contract: %s", payload)
	}
}

package adminauth

import "testing"

func TestClaimsFromMapUsesStableSubjectAndClosedRoleClaim(t *testing.T) {
	claims, err := claimsFromMap(map[string]any{
		"iss": "https://id.example.test", "sub": "subject-1", "nonce": "nonce-1",
		"name": "Ada", "groups": []any{"viewers", "operators"},
	}, "https://id.example.test", "groups")
	if err != nil {
		t.Fatalf("claimsFromMap: %v", err)
	}
	if claims.Subject != "subject-1" || claims.Label != "Ada" || len(claims.RoleValues) != 2 {
		t.Fatalf("claims=%+v", claims)
	}
}

func TestClaimsFromMapRejectsMalformedRoleClaim(t *testing.T) {
	for _, value := range []any{nil, 42, []any{}, []any{"operators", 42}} {
		if _, err := claimsFromMap(map[string]any{
			"sub": "subject-1", "nonce": "nonce-1", "groups": value,
		}, "https://id.example.test", "groups"); err == nil {
			t.Fatalf("malformed role claim accepted: %#v", value)
		}
	}
}

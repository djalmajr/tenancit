package service

import "testing"

func TestNormalizeTenantSlug(t *testing.T) {
	for _, value := range []string{"mydesk", "my-desk", "tenant23"} {
		if got, ok := NormalizeTenantSlug(value); !ok || got != value {
			t.Fatalf("NormalizeTenantSlug(%q) = %q, %v", value, got, ok)
		}
	}
	for _, value := range []string{"MyDesk", "my_desk", "mydesk|", "-mydesk", "mydesk-", "my--desk", ""} {
		if got, ok := NormalizeTenantSlug(value); ok {
			t.Fatalf("NormalizeTenantSlug(%q) unexpectedly accepted as %q", value, got)
		}
	}
}

func TestNormalizeDefinitionKey(t *testing.T) {
	for _, value := range []string{"postgres", "postgres_primary", "postgres-primary"} {
		if got, ok := NormalizeDefinitionKey(value); !ok || got != value {
			t.Fatalf("NormalizeDefinitionKey(%q) = %q, %v", value, got, ok)
		}
	}
	for _, value := range []string{"Postgres", "2postgres", "postgres.main", "postgres|", ""} {
		if got, ok := NormalizeDefinitionKey(value); ok {
			t.Fatalf("NormalizeDefinitionKey(%q) unexpectedly accepted as %q", value, got)
		}
	}
}

func TestValidResourceValue(t *testing.T) {
	for _, test := range []struct {
		dataType string
		value    string
		want     bool
	}{
		{"int", "-42", true}, {"int", "4.2", false},
		{"bool", "true", true}, {"bool", "yes", false},
		{"string", "anything", true},
	} {
		if got := ValidResourceValue(test.dataType, test.value); got != test.want {
			t.Fatalf("ValidResourceValue(%q, %q) = %v, want %v", test.dataType, test.value, got, test.want)
		}
	}
}

package service

import (
	"strings"
	"testing"
)

func TestHashAPIKeyIsDeterministicAndDistinct(t *testing.T) {
	a := HashAPIKey("tnc_alpha")
	if a != HashAPIKey("tnc_alpha") {
		t.Fatal("same API key produced different hashes")
	}
	if a == HashAPIKey("tnc_beta") {
		t.Fatal("different API keys produced the same hash")
	}
	if len(a) != 64 {
		t.Fatalf("hash length = %d, want 64 hex characters", len(a))
	}
}

func TestGenerateAPITokenIsPrefixedAndRandom(t *testing.T) {
	a, err := GenerateAPIToken()
	if err != nil {
		t.Fatalf("GenerateAPIToken: %v", err)
	}
	b, err := GenerateAPIToken()
	if err != nil {
		t.Fatalf("GenerateAPIToken: %v", err)
	}
	if !strings.HasPrefix(a, "tnc_") || !strings.HasPrefix(b, "tnc_") {
		t.Fatalf("tokens do not use tnc_ prefix")
	}
	if a == b {
		t.Fatal("two generated API tokens were identical")
	}
}

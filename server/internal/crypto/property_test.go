package crypto

import (
	"testing"
	"testing/quick"
)

// Property: for ANY plaintext, Decrypt(Encrypt(x)) == x (RN-04/RN-05 round-trip).
// Mutation captured: any corruption of nonce handling or GCM seal/open
// (e.g. reusing a fixed nonce, truncating the tag) breaks the round-trip for
// some input and quick will shrink to a counterexample.
func TestProperty_RoundTrip(t *testing.T) {
	c, err := New(map[int][]byte{1: testKey()}, 1)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	f := func(plain string) bool {
		enc, err := c.Encrypt(plain)
		if err != nil {
			return false
		}
		got, err := c.Decrypt(enc)
		return err == nil && got == plain
	}
	if err := quick.Check(f, &quick.Config{MaxCount: 500}); err != nil {
		t.Fatalf("round-trip property failed: %v", err)
	}
}

// Property: ciphertext never equals the plaintext bytes (no accidental passthrough),
// for any non-empty input.
// Mutation captured: replacing Seal with an identity/no-op cipher would make
// cipher == plaintext and fail here.
func TestProperty_CiphertextDiffersFromPlaintext(t *testing.T) {
	c, _ := New(map[int][]byte{1: testKey()}, 1)
	f := func(plain string) bool {
		if plain == "" {
			return true
		}
		enc, err := c.Encrypt(plain)
		if err != nil {
			return false
		}
		return string(enc.Cipher) != plain
	}
	if err := quick.Check(f, &quick.Config{MaxCount: 500}); err != nil {
		t.Fatalf("ciphertext-distinct property failed: %v", err)
	}
}

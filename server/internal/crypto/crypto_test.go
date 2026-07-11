package crypto

import (
	"bytes"
	"encoding/base64"
	"strings"
	"testing"
)

func testKey() []byte {
	k := make([]byte, 32)
	for i := range k {
		k[i] = byte(i)
	}
	return k
}

func TestRoundTrip(t *testing.T) {
	c, err := New(map[int][]byte{1: testKey()}, 1)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	plain := "demo-secret-value"
	enc, err := c.Encrypt(plain)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if enc.KeyVersion != 1 || len(enc.Nonce) == 0 || len(enc.Cipher) == 0 {
		t.Fatalf("bad enc: %+v", enc)
	}
	if bytes.Contains(enc.Cipher, []byte(plain)) {
		t.Fatal("ciphertext leaks plaintext")
	}
	got, err := c.Decrypt(enc)
	if err != nil || got != plain {
		t.Fatalf("Decrypt = %q err=%v", got, err)
	}
}

func TestTamperDetected(t *testing.T) {
	c, _ := New(map[int][]byte{1: testKey()}, 1)
	enc, _ := c.Encrypt("hello")
	enc.Cipher[0] ^= 0xFF
	if _, err := c.Decrypt(enc); err == nil {
		t.Fatal("expected auth failure on tampered ciphertext")
	}
}

func TestNonceIsRandom(t *testing.T) {
	c, _ := New(map[int][]byte{1: testKey()}, 1)
	a, _ := c.Encrypt("x")
	b, _ := c.Encrypt("x")
	if bytes.Equal(a.Nonce, b.Nonce) {
		t.Fatal("nonces must differ")
	}
}

func TestRejectsBadKeySize(t *testing.T) {
	if _, err := New(map[int][]byte{1: make([]byte, 16)}, 1); err == nil {
		t.Fatal("expected error for non-32-byte key")
	}
}

func TestCrossVersionDecrypt(t *testing.T) {
	k1 := testKey()
	k2 := make([]byte, 32)
	for i := range k2 {
		k2[i] = byte(255 - i)
	}
	cOld, _ := New(map[int][]byte{1: k1}, 1)
	enc, _ := cOld.Encrypt("legacy")
	c, _ := New(map[int][]byte{1: k1, 2: k2}, 2)
	got, err := c.Decrypt(enc)
	if err != nil || got != "legacy" {
		t.Fatalf("cross-version decrypt: got=%q err=%v", got, err)
	}
}

func TestByteAPIExposesOnlyVersionMetadataAndRoundTrips(t *testing.T) {
	c, err := New(map[int][]byte{1: testKey()}, 1)
	if err != nil {
		t.Fatal(err)
	}
	plaintext := []byte("mutable-operational-secret")
	encrypted, err := c.EncryptBytes(plaintext)
	if err != nil {
		t.Fatal(err)
	}
	opened, err := c.DecryptBytes(encrypted)
	if err != nil || !bytes.Equal(opened, plaintext) {
		t.Fatalf("DecryptBytes mismatch err=%v", err)
	}
	if c.CurrentVersion() != 1 || !c.HasVersion(1) || c.HasVersion(2) {
		t.Fatal("unexpected version metadata")
	}
	if nonceSize, ok := c.NonceSize(1); !ok || nonceSize != len(encrypted.Nonce) {
		t.Fatalf("nonce metadata = %d, %v", nonceSize, ok)
	}
	if overhead, ok := c.Overhead(1); !ok || overhead <= 0 || len(encrypted.Cipher) < overhead {
		t.Fatalf("overhead metadata = %d, %v", overhead, ok)
	}
}

func TestFromEnvRequiresCurrentKey(t *testing.T) {
	t.Setenv("TENANCIT_AES_KEY", "")
	t.Setenv("TENANCIT_AES_KEY_VERSION", "")

	_, err := FromEnv()
	if err == nil || !strings.Contains(err.Error(), "TENANCIT_AES_KEY is required") {
		t.Fatalf("FromEnv error = %v, want required-key error", err)
	}
}

func TestFromEnvRejectsInvalidBase64(t *testing.T) {
	t.Setenv("TENANCIT_AES_KEY", "not-base64")
	t.Setenv("TENANCIT_AES_KEY_VERSION", "1")

	_, err := FromEnv()
	if err == nil || !strings.Contains(err.Error(), "not valid base64") {
		t.Fatalf("FromEnv error = %v, want base64 error", err)
	}
}

func TestFromEnvRejectsNonCanonicalCurrentVersion(t *testing.T) {
	for _, value := range []string{"0", "-1", "02", "2garbage"} {
		t.Run(value, func(t *testing.T) {
			t.Setenv("TENANCIT_AES_KEY", base64.StdEncoding.EncodeToString(testKey()))
			t.Setenv("TENANCIT_AES_KEY_VERSION", value)

			_, err := FromEnv()
			if err == nil || !strings.Contains(err.Error(), "invalid TENANCIT_AES_KEY_VERSION") {
				t.Fatalf("FromEnv version %q error = %v, want canonical-positive-version error", value, err)
			}
		})
	}
}

func TestFromEnvAcceptsMaximumStoredKeyVersion(t *testing.T) {
	t.Setenv("TENANCIT_AES_KEY", base64.StdEncoding.EncodeToString(testKey()))
	t.Setenv("TENANCIT_AES_KEY_VERSION", "2147483647")

	c, err := FromEnv()
	if err != nil {
		t.Fatalf("FromEnv max stored version: %v", err)
	}
	enc, err := c.Encrypt("boundary")
	if err != nil {
		t.Fatalf("Encrypt max stored version: %v", err)
	}
	if enc.KeyVersion != 2147483647 {
		t.Fatalf("Encrypt key version = %d, want 2147483647", enc.KeyVersion)
	}
}

func TestFromEnvRejectsCurrentVersionAboveStorageRange(t *testing.T) {
	t.Setenv("TENANCIT_AES_KEY", base64.StdEncoding.EncodeToString(testKey()))
	t.Setenv("TENANCIT_AES_KEY_VERSION", "2147483648")

	_, err := FromEnv()
	if err == nil || !strings.Contains(err.Error(), "invalid TENANCIT_AES_KEY_VERSION") {
		t.Fatalf("FromEnv out-of-range current version error = %v", err)
	}
}

func TestFromEnvRejectsVersionedKeyAboveStorageRange(t *testing.T) {
	t.Setenv("TENANCIT_AES_KEY", base64.StdEncoding.EncodeToString(testKey()))
	t.Setenv("TENANCIT_AES_KEY_VERSION", "1")
	t.Setenv("TENANCIT_AES_KEY_V2147483648", base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{0x2A}, 32)))

	_, err := FromEnv()
	if err == nil || !strings.Contains(err.Error(), "invalid versioned key variable") {
		t.Fatalf("FromEnv out-of-range versioned key error = %v", err)
	}
}

func TestFromEnvRejectsNonCanonicalVersionedKeyName(t *testing.T) {
	t.Setenv("TENANCIT_AES_KEY", base64.StdEncoding.EncodeToString(testKey()))
	t.Setenv("TENANCIT_AES_KEY_VERSION", "1")
	t.Setenv("TENANCIT_AES_KEY_V02", base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{0x2A}, 32)))

	_, err := FromEnv()
	if err == nil || !strings.Contains(err.Error(), "invalid versioned key variable") {
		t.Fatalf("FromEnv error = %v, want non-canonical-versioned-key error", err)
	}
}

func TestFromEnvBuildsCurrentCryptor(t *testing.T) {
	t.Setenv("TENANCIT_AES_KEY", base64.StdEncoding.EncodeToString(testKey()))
	t.Setenv("TENANCIT_AES_KEY_VERSION", "1")

	c, err := FromEnv()
	if err != nil {
		t.Fatalf("FromEnv: %v", err)
	}
	enc, err := c.Encrypt("configured-secret")
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	got, err := c.Decrypt(enc)
	if err != nil || got != "configured-secret" {
		t.Fatalf("Decrypt = %q, err=%v", got, err)
	}
}

func TestFromEnvLoadsOlderVersionsForDecryption(t *testing.T) {
	oldKey := testKey()
	newKey := bytes.Repeat([]byte{0xA5}, 32)
	oldCryptor, err := New(map[int][]byte{1: oldKey}, 1)
	if err != nil {
		t.Fatalf("New old cryptor: %v", err)
	}
	ciphertext, err := oldCryptor.Encrypt("legacy-secret")
	if err != nil {
		t.Fatalf("Encrypt old value: %v", err)
	}

	t.Setenv("TENANCIT_AES_KEY", base64.StdEncoding.EncodeToString(newKey))
	t.Setenv("TENANCIT_AES_KEY_VERSION", "2")
	t.Setenv("TENANCIT_AES_KEY_V1", base64.StdEncoding.EncodeToString(oldKey))

	c, err := FromEnv()
	if err != nil {
		t.Fatalf("FromEnv: %v", err)
	}
	got, err := c.Decrypt(ciphertext)
	if err != nil || got != "legacy-secret" {
		t.Fatalf("Decrypt legacy = %q, err=%v", got, err)
	}
}

func TestFromEnvLoadsStagedFutureVersionForRollingCutover(t *testing.T) {
	currentKey := testKey()
	futureKey := bytes.Repeat([]byte{0x5A}, 32)
	futureCryptor, err := New(map[int][]byte{2: futureKey}, 2)
	if err != nil {
		t.Fatalf("New future cryptor: %v", err)
	}
	futureCiphertext, err := futureCryptor.Encrypt("written-by-upgraded-replica")
	if err != nil {
		t.Fatalf("Encrypt future value: %v", err)
	}

	t.Setenv("TENANCIT_AES_KEY", base64.StdEncoding.EncodeToString(currentKey))
	t.Setenv("TENANCIT_AES_KEY_VERSION", "1")
	t.Setenv("TENANCIT_AES_KEY_V2", base64.StdEncoding.EncodeToString(futureKey))

	c, err := FromEnv()
	if err != nil {
		t.Fatalf("FromEnv: %v", err)
	}
	got, err := c.Decrypt(futureCiphertext)
	if err != nil || got != "written-by-upgraded-replica" {
		t.Fatalf("Decrypt staged future value = %q, err=%v", got, err)
	}
	currentCiphertext, err := c.Encrypt("still-current")
	if err != nil {
		t.Fatalf("Encrypt current value: %v", err)
	}
	if currentCiphertext.KeyVersion != 1 {
		t.Fatalf("Encrypt key version = %d, want current version 1", currentCiphertext.KeyVersion)
	}
}

func TestFromEnvRejectsConflictingCurrentVersionAlias(t *testing.T) {
	t.Setenv("TENANCIT_AES_KEY", base64.StdEncoding.EncodeToString(testKey()))
	t.Setenv("TENANCIT_AES_KEY_VERSION", "2")
	t.Setenv("TENANCIT_AES_KEY_V2", base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{0x7B}, 32)))

	_, err := FromEnv()
	if err == nil || !strings.Contains(err.Error(), "conflicts with TENANCIT_AES_KEY") {
		t.Fatalf("FromEnv error = %v, want conflicting-current-key error", err)
	}
}

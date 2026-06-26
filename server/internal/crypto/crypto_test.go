package crypto

import (
	"bytes"
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

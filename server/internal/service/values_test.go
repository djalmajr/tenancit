package service

import (
	"testing"

	"github.com/centralit/resource-tenant/server/internal/crypto"
	"github.com/centralit/resource-tenant/server/internal/store/db"
)

func newCryptor(t *testing.T) *crypto.Cryptor {
	t.Helper()
	k := make([]byte, 32)
	for i := range k {
		k[i] = byte(i + 1)
	}
	c, err := crypto.New(map[int][]byte{1: k}, 1)
	if err != nil {
		t.Fatalf("crypto.New: %v", err)
	}
	return c
}

func TestEncodeValue_NonSecret_StoresPlain(t *testing.T) {
	c := newCryptor(t)
	p, err := encodeValue(c, false, "db-host.internal")
	if err != nil {
		t.Fatal(err)
	}
	if p.ValuePlain == nil || *p.ValuePlain != "db-host.internal" {
		t.Fatalf("expected plain stored, got %+v", p)
	}
	if p.ValueCipher != nil {
		t.Fatal("non-secret must not be encrypted")
	}
}

func TestEncodeValue_Secret_Encrypts(t *testing.T) {
	c := newCryptor(t)
	p, err := encodeValue(c, true, "super-secret")
	if err != nil {
		t.Fatal(err)
	}
	if p.ValuePlain != nil {
		t.Fatal("secret must not be plain")
	}
	if len(p.ValueCipher) == 0 || len(p.Nonce) == 0 || p.KeyVersion == nil {
		t.Fatalf("secret must have cipher+nonce+key_version, got %+v", p)
	}
}

func TestPresentValue_SecretMaskedByDefault(t *testing.T) {
	c := newCryptor(t)
	p, _ := encodeValue(c, true, "hunter2")
	stored := db.TenantResourceValue{ValueCipher: p.ValueCipher, Nonce: p.Nonce, KeyVersion: p.KeyVersion}
	masked, _ := presentValue(c, true, stored, false)
	if masked != MaskedValue {
		t.Fatalf("want masked, got %q", masked)
	}
	revealed, err := presentValue(c, true, stored, true)
	if err != nil || revealed != "hunter2" {
		t.Fatalf("reveal: got %q err=%v", revealed, err)
	}
}

func TestDecodeValue_ConsumerRoundTrip(t *testing.T) {
	c := newCryptor(t)
	p, _ := encodeValue(c, true, "pg-pass")
	stored := db.TenantResourceValue{ValueCipher: p.ValueCipher, Nonce: p.Nonce, KeyVersion: p.KeyVersion}
	got, err := decodeValue(c, stored)
	if err != nil || got != "pg-pass" {
		t.Fatalf("decode = %q err=%v", got, err)
	}
}

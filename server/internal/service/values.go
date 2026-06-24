package service

import (
	"github.com/djalmajr/konvario/server/internal/crypto"
	"github.com/djalmajr/konvario/server/internal/store/db"
)

// MaskedValue is the masked placeholder returned by admin endpoints (RN-06).
const MaskedValue = "••••••••"

// encodeValue turns a plaintext into UpsertResourceValueParams, encrypting when
// the field is secret (RN-04). Caller fills TenantResourceID/ResourceFieldID.
func encodeValue(c *crypto.Cryptor, isSecret bool, plaintext string) (db.UpsertResourceValueParams, error) {
	if !isSecret {
		p := plaintext
		return db.UpsertResourceValueParams{ValuePlain: &p}, nil
	}
	enc, err := c.Encrypt(plaintext)
	if err != nil {
		return db.UpsertResourceValueParams{}, err
	}
	kv := int32(enc.KeyVersion)
	return db.UpsertResourceValueParams{
		ValueCipher: enc.Cipher,
		Nonce:       enc.Nonce,
		KeyVersion:  &kv,
	}, nil
}

// decodeValue returns the cleartext value (RN-05). For secret values it decrypts.
func decodeValue(c *crypto.Cryptor, v db.TenantResourceValue) (string, error) {
	if v.ValueCipher == nil {
		if v.ValuePlain != nil {
			return *v.ValuePlain, nil
		}
		return "", nil
	}
	kv := 0
	if v.KeyVersion != nil {
		kv = int(*v.KeyVersion)
	}
	return c.Decrypt(crypto.Encrypted{Cipher: v.ValueCipher, Nonce: v.Nonce, KeyVersion: kv})
}

// presentValue masks secret values unless reveal is true (RN-06).
func presentValue(c *crypto.Cryptor, isSecret bool, v db.TenantResourceValue, reveal bool) (string, error) {
	if isSecret && !reveal {
		return MaskedValue, nil
	}
	return decodeValue(c, v)
}

// Package crypto provides AES-256-GCM encryption for sensitive field values,
// with an externalized key and key-version support for rotation (RN-04/RN-05).
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
)

// Encrypted is the stored form of a secret value.
type Encrypted struct {
	Cipher     []byte
	Nonce      []byte
	KeyVersion int
}

// Cryptor encrypts/decrypts using AES-256-GCM with versioned keys.
type Cryptor struct {
	aeads   map[int]cipher.AEAD
	current int
}

// New builds a Cryptor from a map of version->32-byte key and the current
// version used for new encryptions. All keys must be exactly 32 bytes.
func New(keys map[int][]byte, current int) (*Cryptor, error) {
	if len(keys) == 0 {
		return nil, errors.New("crypto: no keys provided")
	}
	if _, ok := keys[current]; !ok {
		return nil, fmt.Errorf("crypto: current key version %d not present", current)
	}
	aeads := make(map[int]cipher.AEAD, len(keys))
	for v, key := range keys {
		if len(key) != 32 {
			return nil, fmt.Errorf("crypto: key v%d must be 32 bytes, got %d", v, len(key))
		}
		block, err := aes.NewCipher(key)
		if err != nil {
			return nil, fmt.Errorf("crypto: new cipher v%d: %w", v, err)
		}
		aead, err := cipher.NewGCM(block)
		if err != nil {
			return nil, fmt.Errorf("crypto: new gcm v%d: %w", v, err)
		}
		aeads[v] = aead
	}
	return &Cryptor{aeads: aeads, current: current}, nil
}

// Encrypt seals the plaintext with the current key version and a fresh nonce.
func (c *Cryptor) Encrypt(plaintext string) (Encrypted, error) {
	aead := c.aeads[c.current]
	nonce := make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return Encrypted{}, fmt.Errorf("crypto: nonce: %w", err)
	}
	ct := aead.Seal(nil, nonce, []byte(plaintext), nil)
	return Encrypted{Cipher: ct, Nonce: nonce, KeyVersion: c.current}, nil
}

// Decrypt opens an Encrypted value using its recorded key version.
func (c *Cryptor) Decrypt(e Encrypted) (string, error) {
	aead, ok := c.aeads[e.KeyVersion]
	if !ok {
		return "", fmt.Errorf("crypto: unknown key version %d", e.KeyVersion)
	}
	pt, err := aead.Open(nil, e.Nonce, e.Cipher, nil)
	if err != nil {
		return "", fmt.Errorf("crypto: open: %w", err)
	}
	return string(pt), nil
}

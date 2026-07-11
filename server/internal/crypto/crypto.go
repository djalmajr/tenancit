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
	buffer := []byte(plaintext)
	encrypted, err := c.EncryptBytes(buffer)
	clear(buffer)
	return encrypted, err
}

// EncryptBytes seals a caller-owned buffer without converting it to a string.
// Callers handling operational plaintext can zero their input after use.
func (c *Cryptor) EncryptBytes(plaintext []byte) (Encrypted, error) {
	aead := c.aeads[c.current]
	nonce := make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return Encrypted{}, fmt.Errorf("crypto: nonce: %w", err)
	}
	ct := aead.Seal(nil, nonce, plaintext, nil)
	return Encrypted{Cipher: ct, Nonce: nonce, KeyVersion: c.current}, nil
}

// Decrypt opens an Encrypted value using its recorded key version.
func (c *Cryptor) Decrypt(e Encrypted) (string, error) {
	plaintext, err := c.DecryptBytes(e)
	if err != nil {
		return "", err
	}
	value := string(plaintext)
	clear(plaintext)
	return value, nil
}

// DecryptBytes authenticates and opens a value into a mutable buffer.
func (c *Cryptor) DecryptBytes(e Encrypted) ([]byte, error) {
	aead, ok := c.aeads[e.KeyVersion]
	if !ok {
		return nil, fmt.Errorf("crypto: unknown key version %d", e.KeyVersion)
	}
	pt, err := aead.Open(nil, e.Nonce, e.Cipher, nil)
	if err != nil {
		return nil, fmt.Errorf("crypto: open: %w", err)
	}
	return pt, nil
}

// CurrentVersion returns metadata only; key material remains private.
func (c *Cryptor) CurrentVersion() int { return c.current }

// HasVersion reports whether a version can be decrypted.
func (c *Cryptor) HasVersion(version int) bool {
	_, ok := c.aeads[version]
	return ok
}

// NonceSize returns the required nonce length for a loaded key version.
func (c *Cryptor) NonceSize(version int) (int, bool) {
	aead, ok := c.aeads[version]
	if !ok {
		return 0, false
	}
	return aead.NonceSize(), true
}

// Overhead returns the minimum authentication overhead for a loaded version.
func (c *Cryptor) Overhead(version int) (int, bool) {
	aead, ok := c.aeads[version]
	if !ok {
		return 0, false
	}
	return aead.Overhead(), true
}

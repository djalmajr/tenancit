package service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
)

// HashAPIKey returns the hex SHA-256 of a service token. Only the hash is
// stored (api_clients.key_hash); the raw token is shown once at creation (RN-09).
func HashAPIKey(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// GenerateAPIToken returns a fresh opaque service token (shown once).
func GenerateAPIToken() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "tnc_" + hex.EncodeToString(b), nil
}

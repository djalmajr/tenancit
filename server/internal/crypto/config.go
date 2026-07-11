package crypto

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"os"
	"strconv"
	"strings"
)

const maxStoredKeyVersion = 1<<31 - 1

// FromEnv builds a Cryptor from environment configuration.
//
//	TENANCIT_AES_KEY         base64 (std) of a 32-byte key for the current version
//	TENANCIT_AES_KEY_VERSION integer current key version (default 1)
//	TENANCIT_AES_KEY_V<n>    optional additional keys (base64) for rotation
func FromEnv() (*Cryptor, error) {
	version := 1
	if v := os.Getenv("TENANCIT_AES_KEY_VERSION"); v != "" {
		parsedVersion, err := strconv.Atoi(v)
		if err != nil {
			return nil, fmt.Errorf("crypto: invalid TENANCIT_AES_KEY_VERSION: %w", err)
		}
		if parsedVersion <= 0 || parsedVersion > maxStoredKeyVersion || strconv.Itoa(parsedVersion) != v {
			return nil, fmt.Errorf("crypto: invalid TENANCIT_AES_KEY_VERSION: must be a canonical positive integer")
		}
		version = parsedVersion
	}

	keys := map[int][]byte{}

	cur := os.Getenv("TENANCIT_AES_KEY")
	if cur == "" {
		return nil, fmt.Errorf("crypto: TENANCIT_AES_KEY is required")
	}
	k, err := base64.StdEncoding.DecodeString(cur)
	if err != nil {
		return nil, fmt.Errorf("crypto: TENANCIT_AES_KEY not valid base64: %w", err)
	}
	keys[version] = k

	const versionedKeyPrefix = "TENANCIT_AES_KEY_V"
	for _, entry := range os.Environ() {
		name, raw, ok := strings.Cut(entry, "=")
		if !ok || name == "TENANCIT_AES_KEY_VERSION" || !strings.HasPrefix(name, versionedKeyPrefix) {
			continue
		}
		versionText := strings.TrimPrefix(name, versionedKeyPrefix)
		keyVersion, err := strconv.Atoi(versionText)
		if err != nil || keyVersion <= 0 || keyVersion > maxStoredKeyVersion || strconv.Itoa(keyVersion) != versionText {
			return nil, fmt.Errorf("crypto: invalid versioned key variable %s", name)
		}
		additionalKey, err := base64.StdEncoding.DecodeString(raw)
		if err != nil {
			return nil, fmt.Errorf("crypto: %s not valid base64: %w", name, err)
		}
		if keyVersion == version && !bytes.Equal(additionalKey, k) {
			return nil, fmt.Errorf("crypto: %s conflicts with TENANCIT_AES_KEY", name)
		}
		keys[keyVersion] = additionalKey
	}

	return New(keys, version)
}

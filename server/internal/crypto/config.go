package crypto

import (
	"encoding/base64"
	"fmt"
	"os"
)

// FromEnv builds a Cryptor from environment configuration.
//
//	RT_AES_KEY         base64 (std) of a 32-byte key for the current version
//	RT_AES_KEY_VERSION integer current key version (default 1)
//	RT_AES_KEY_V<n>    optional older keys (base64) for rotation
func FromEnv() (*Cryptor, error) {
	version := 1
	if v := os.Getenv("RT_AES_KEY_VERSION"); v != "" {
		if _, err := fmt.Sscanf(v, "%d", &version); err != nil {
			return nil, fmt.Errorf("crypto: invalid RT_AES_KEY_VERSION: %w", err)
		}
	}

	keys := map[int][]byte{}

	cur := os.Getenv("RT_AES_KEY")
	if cur == "" {
		return nil, fmt.Errorf("crypto: RT_AES_KEY is required")
	}
	k, err := base64.StdEncoding.DecodeString(cur)
	if err != nil {
		return nil, fmt.Errorf("crypto: RT_AES_KEY not valid base64: %w", err)
	}
	keys[version] = k

	for v := 1; v < version; v++ {
		if raw := os.Getenv(fmt.Sprintf("RT_AES_KEY_V%d", v)); raw != "" {
			old, err := base64.StdEncoding.DecodeString(raw)
			if err != nil {
				return nil, fmt.Errorf("crypto: RT_AES_KEY_V%d not valid base64: %w", v, err)
			}
			keys[v] = old
		}
	}

	return New(keys, version)
}

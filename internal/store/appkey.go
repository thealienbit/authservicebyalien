package store

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

// GenerateAppKey returns the plaintext appkey (shown to the operator once) and
// the SHA-256 hash that is persisted. The plaintext is never stored.
func GenerateAppKey() (plain string, hash string, err error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", "", fmt.Errorf("store: generate app key: %w", err)
	}

	plain = base64.RawURLEncoding.EncodeToString(raw)
	return plain, HashAppKey(plain), nil
}

func HashAppKey(plain string) string {
	return HashToken(plain)
}

// HashToken is the SHA-256/base64url scheme used for every opaque token stored
// at rest (app keys, refresh tokens, portal sessions).
func HashToken(plain string) string {
	sum := sha256.Sum256([]byte(plain))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

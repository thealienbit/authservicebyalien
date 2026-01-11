package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
)

func GenerateToken() (plain string, hash string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return "", "", err
	}

	plain = base64.RawURLEncoding.EncodeToString(b)
	h := sha256.Sum256([]byte(plain))
	hash = base64.RawURLEncoding.EncodeToString(h[:])

	return
}

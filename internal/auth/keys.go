package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"errors"
	"math/big"
)

type SigningKey struct {
	KeyID string
	PrivateKey *rsa.PrivateKey
	PublicKey *rsa.PublicKey
}

func GenerateRSAKey(keyID string) (*SigningKey, error) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	return &SigningKey{
		KeyID: keyID,
		PrivateKey: priv,
		PublicKey: &priv.PublicKey,
	}, nil
}

func publicKeyToJWKS(key *SigningKey) map[string]string {
	n := base64.RawURLEncoding.EncodeToString(key.PublicKey.N.Bytes())
	e := base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.PublicKey.E)).Bytes())
	
	return map[string]string{
		"kty": "RSA",
		"kid": key.KeyID,
		"use": "sig",
		"alg": "RS256",
		"n": n,
		"e": e,
	}
}


func vaidateKey(k *SigningKey) error {
	if k == nil || k.PrivateKey == nil || k.PublicKey == nil {
		return errors.New("Invalid sigining key")
	}
	return nil
}

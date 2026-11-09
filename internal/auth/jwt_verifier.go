package auth

import (
	"crypto/rsa"
	"errors"

	"github.com/golang-jwt/jwt/v5"

	"github.com/thealienbit/authservicebyalien/internal/signing"
)

type JWTVerifier struct {
	Issuer   string
	Audience string
	Keys     map[string]*rsa.PublicKey // kid → key
}

func NewJWTVerifier(
	issuer string,
	audience string,
	signingKeys []*signing.Key,
) *JWTVerifier {

	keys := make(map[string]*rsa.PublicKey)
	for _, k := range signingKeys {
		keys[k.KeyID] = k.PublicKey
	}

	return &JWTVerifier{
		Issuer:   issuer,
		Audience: audience,
		Keys:     keys,
	}
}

func (v *JWTVerifier) keyFunc(token *jwt.Token) (interface{}, error) {
	if token.Method.Alg() != jwt.SigningMethodRS256.Alg() {
		return nil, errors.New("unexpected signing method")
	}

	kid, ok := token.Header["kid"].(string)
	if !ok {
		return nil, errors.New("missing kid")
	}

	key, ok := v.Keys[kid]
	if !ok {
		return nil, errors.New("unknown signing key")
	}

	return key, nil
}


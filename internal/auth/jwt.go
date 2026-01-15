package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type JWTIssuer struct {
	Issuer string
	Audience string
	SigningKey *SigningKey
}

func NewJWTIssuer(issuer, audience string, key *SigningKey) (*JWTIssuer, error) {
	if err := vaidateKey(key); err != nil {
		return nil, err
	}

	return &JWTIssuer{
		Issuer: issuer,
		Audience: audience,
		SigningKey: key,
	}, nil
}


func (j *JWTIssuer) IssueAccessToken(
	userID uuid.UUID,
	email string,
	emailVerified bool,
) (string, error) {

	now := time.Now()


	claims := jwt.MapClaims{
		"sub":            userID.String(),
		"iss":            j.Issuer,
		"aud":            j.Audience,
		"email":          email,
		"email_verified": emailVerified,
		"iat":            now.Unix(),
		"exp":            now.Add(15 * time.Minute).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = j.SigningKey.KeyID

	return token.SignedString(j.SigningKey.PrivateKey)
}


const AccessTokenTTLSeconds = 900

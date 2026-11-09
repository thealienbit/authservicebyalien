package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/thealienbit/authservicebyalien/internal/signing"
	"github.com/thealienbit/authservicebyalien/internal/store"
)

const AccessTokenTTLSeconds = 900

type JWTIssuer struct {
	Issuer     string
	SigningKey *signing.Key
}

type AccessTokenClaims struct {
	UserID        uuid.UUID
	Email         string
	EmailVerified bool
	AppID         uuid.UUID
	OrgID         uuid.UUID
	Role          store.Role
}

func NewJWTIssuer(issuer string, key *signing.Key) (*JWTIssuer, error) {
	if err := key.Validate(); err != nil {
		return nil, err
	}

	return &JWTIssuer{
		Issuer:     issuer,
		SigningKey: key,
	}, nil
}

func (j *JWTIssuer) IssueAccessToken(claims AccessTokenClaims) (string, error) {
	now := time.Now()

	mapClaims := jwt.MapClaims{
		"sub":            claims.UserID.String(),
		"iss":            j.Issuer,
		"aud":            claims.AppID.String(),
		"email":          claims.Email,
		"email_verified": claims.EmailVerified,
		"org":            claims.OrgID.String(),
		"role":           string(claims.Role),
		"iat":            now.Unix(),
		"exp":            now.Add(AccessTokenTTLSeconds * time.Second).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, mapClaims)
	token.Header["kid"] = j.SigningKey.KeyID

	return token.SignedString(j.SigningKey.PrivateKey)
}

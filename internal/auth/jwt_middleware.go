package auth

import (
	"net/http"

	"github.com/golang-jwt/jwt/v5"
)

func JWTMiddleware(verifier *JWTVerifier) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			authHeader := r.Header.Get("Authorization")
			if authHeader == "" || len(authHeader) < 8 {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			// Expect: "Bearer <token>"
			tokenString := authHeader[7:]

			options := []jwt.ParserOption{
				jwt.WithIssuer(verifier.Issuer),
				jwt.WithValidMethods([]string{"RS256"}),
				jwt.WithExpirationRequired(),
			}
			// Tokens are audiences per app. Auth's own endpoints only need the
			// subject, so audience validation is opt-in (empty = skip).
			if verifier.Audience != "" {
				options = append(options, jwt.WithAudience(verifier.Audience))
			}

			claims := jwt.MapClaims{}
			token, err := jwt.ParseWithClaims(
				tokenString,
				claims,
				verifier.keyFunc,
				options...,
			)

			if err != nil || !token.Valid {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			userID, ok := claims["sub"].(string)
			if !ok {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			email, _ := claims["email"].(string)
			emailVerified, _ := claims["email_verified"].(bool)
			orgID, _ := claims["org"].(string)
			role, _ := claims["role"].(string)

			ctx := WithAuthContext(
				r.Context(),
				userID,
				email,
				emailVerified,
				orgID,
				role,
			)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

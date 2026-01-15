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

			claims := jwt.MapClaims{}
			token, err := jwt.ParseWithClaims(
				tokenString,
				claims,
				verifier.keyFunc,
				jwt.WithAudience(verifier.Audience),
				jwt.WithIssuer(verifier.Issuer),
				jwt.WithValidMethods([]string{"RS256"}),
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

			ctx := WithAuthContext(
				r.Context(),
				userID,
				email,
				emailVerified,
			)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}


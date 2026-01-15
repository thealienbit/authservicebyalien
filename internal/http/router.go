package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"

	authHandler "github.com/thealienbit/authservicebyalien/internal/auth"
	"github.com/thealienbit/authservicebyalien/internal/health"

	"github.com/thealienbit/authservicebyalien/internal/email"
)

func NewRouter(db *pgxpool.Pool, jwtIssuer *authHandler.JWTIssuer, signingKeys []*authHandler.SigningKey, emailSender email.EmailSender, 
) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(RateLimitMiddleware(GlobalLimiter))

	// Health check (no auth)
	r.Get("/health", health.Handler)

	// JWKS (PUBLIC, no auth)
	r.Get(
		"/auth/v1/.well-known/jwks.json",
		authHandler.JWKSHandler(signingKeys),
	)

	auth := authHandler.NewHandler(db, jwtIssuer, emailSender)


	verifier := authHandler.NewJWTVerifier(
	"auth-service",
	"your-app",
	signingKeys,
)


	r.Route("/auth/v1", func(r chi.Router) {
		r.With(RateLimitMiddleware(RegisterLimiter)).Post("/register", auth.Register)
		r.With(RateLimitMiddleware(EmailLimiter)).Post("/email/verify", auth.VerifyEmail)
		r.With(RateLimitMiddleware(EmailLimiter)).Post("/email/resend", auth.ResendVerification)
		r.With(RateLimitMiddleware(LoginLimiter)).Post("/login", auth.Login)
		r.Post("/token/refresh", auth.RefreshToken)
		r.Post("/logout", auth.Logout)
		r.Post("/password/forgot", auth.ForgotPassword)
		r.Post("/password/reset", auth.ResetPassword)

		r.Group(func(r chi.Router) {
			r.Use(authHandler.JWTMiddleware(verifier))
			r.Post("/logout/all", auth.LogoutAll)
		})
	})

	return r
}

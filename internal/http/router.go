package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"

	authHandler "github.com/thealienbit/authservicebyalien/internal/auth"
	"github.com/thealienbit/authservicebyalien/internal/health"
)

func NewRouter(db *pgxpool.Pool) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Health check (no auth)
	r.Get("/health", health.Handler)

	auth := authHandler.NewHandler(db)

	r.Route("/auth/v1", func(r chi.Router) {
		r.Post("/register", auth.Register)
		r.Post("/email/verify", auth.VerifyEmail)
		r.Post("/email/resend", auth.ResendVerification)
	})

	return r
}

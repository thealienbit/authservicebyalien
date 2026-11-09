package http

import (
	"net"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"

	authHandler "github.com/thealienbit/authservicebyalien/internal/auth"
	"github.com/thealienbit/authservicebyalien/internal/email"
	"github.com/thealienbit/authservicebyalien/internal/health"
	"github.com/thealienbit/authservicebyalien/internal/portal"
	"github.com/thealienbit/authservicebyalien/internal/signing"
	"github.com/thealienbit/authservicebyalien/internal/store"
)

type Deps struct {
	DB             *pgxpool.Pool
	JWTIssuer      *authHandler.JWTIssuer
	SigningKeys    []*signing.Key
	EmailSender    email.EmailSender
	Store          *store.Store
	TrustedProxies []*net.IPNet
	SecureCookies  bool
}

func NewRouter(deps Deps) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	// middleware.RealIP is intentionally not used: it trusts X-Forwarded-For
	// unconditionally. ClientIP handles it against the trusted proxy list.
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	ipKey := func(r *http.Request) string {
		return ClientIP(r, deps.TrustedProxies)
	}
	appIPKey := func(r *http.Request) string {
		app, ok := authHandler.AppFromContext(r.Context())
		if !ok {
			return ipKey(r)
		}
		return app.ID.String() + "|" + ipKey(r)
	}

	// Public routes: no app key.
	r.Group(func(r chi.Router) {
		r.Use(RateLimitMiddleware(GlobalLimiter, ipKey))
		r.Get("/health", health.Handler)
		r.Get("/auth/v1/.well-known/jwks.json", authHandler.JWKSHandler(deps.SigningKeys))
	})

	auth := authHandler.NewHandler(deps.DB, deps.JWTIssuer, deps.EmailSender, deps.Store)

	verifier := authHandler.NewJWTVerifier(
		"auth-service",
		"",
		deps.SigningKeys,
	)

	// App-to-auth endpoints: every one requires a valid X-App-Key.
	r.Route("/auth/v1", func(r chi.Router) {
		r.Use(AppKeyMiddleware(deps.Store))
		r.Use(RateLimitMiddleware(GlobalLimiter, appIPKey))

		r.With(RateLimitMiddleware(RegisterLimiter, appIPKey)).Post("/register", auth.Register)
		r.With(RateLimitMiddleware(EmailLimiter, appIPKey)).Post("/email/verify", auth.VerifyEmail)
		r.With(RateLimitMiddleware(EmailLimiter, appIPKey)).Post("/email/resend", auth.ResendVerification)
		r.With(RateLimitMiddleware(LoginLimiter, appIPKey)).Post("/login", auth.Login)
		r.Post("/token/refresh", auth.RefreshToken)
		r.Post("/logout", auth.Logout)
		r.Post("/password/forgot", auth.ForgotPassword)
		r.Post("/password/reset", auth.ResetPassword)

		r.Group(func(r chi.Router) {
			r.Use(authHandler.JWTMiddleware(verifier))
			r.Post("/logout/all", auth.LogoutAll)
		})
	})

	portalHandler := portal.New(portal.Deps{
		Store:           deps.Store,
		SecureCookies:   deps.SecureCookies,
		ClientIP:        ipKey,
		LoginMiddleware: RateLimitMiddleware(LoginLimiter, ipKey),
	})

	r.Route("/portal", func(r chi.Router) {
		r.Use(RateLimitMiddleware(GlobalLimiter, ipKey))
		portalHandler.Register(r)
	})

	return r
}

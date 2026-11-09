package portal

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"net"
	"net/http"
	"time"

	"github.com/thealienbit/authservicebyalien/internal/store"
)

const (
	sessionCookie = "portal_session"
	sessionTTL    = 12 * time.Hour
)

var errNoSession = errors.New("portal: no valid session")

type sessionContextKey struct{}

func withSession(ctx context.Context, session *store.PortalSession) context.Context {
	return context.WithValue(ctx, sessionContextKey{}, session)
}

func sessionFromContext(ctx context.Context) (*store.PortalSession, bool) {
	session, ok := ctx.Value(sessionContextKey{}).(*store.PortalSession)
	return session, ok
}

func (h *Handler) sessionFromRequest(r *http.Request) (*store.PortalSession, error) {
	cookie, err := r.Cookie(sessionCookie)
	if err != nil {
		return nil, errNoSession
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	session, err := h.store.GetPortalSession(ctx, store.HashToken(cookie.Value))
	if err != nil {
		return nil, err
	}

	if session.RevokedAt != nil || time.Now().After(session.ExpiresAt) {
		return nil, errNoSession
	}

	return session, nil
}

func (h *Handler) requireSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, err := h.sessionFromRequest(r)
		if err != nil {
			http.Redirect(w, r, "/portal/login", http.StatusSeeOther)
			return
		}

		next.ServeHTTP(w, r.WithContext(withSession(r.Context(), session)))
	})
}

func (h *Handler) requireCSRF(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, ok := sessionFromContext(r.Context())
		if !ok {
			http.Redirect(w, r, "/portal/login", http.StatusSeeOther)
			return
		}

		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		provided := r.PostFormValue("csrf")
		if subtle.ConstantTimeCompare([]byte(provided), []byte(session.CSRF)) != 1 {
			http.Error(w, "invalid csrf token", http.StatusForbidden)
			return
		}

		next(w, r)
	}
}

func (h *Handler) setSessionCookie(w http.ResponseWriter, token string, expires time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    token,
		Path:     "/portal",
		HttpOnly: true,
		Secure:   h.secureCookies,
		SameSite: http.SameSiteLaxMode,
		Expires:  expires,
		MaxAge:   int(time.Until(expires).Seconds()),
	})
}

func (h *Handler) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    "",
		Path:     "/portal",
		HttpOnly: true,
		Secure:   h.secureCookies,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})
}

func (h *Handler) revokeRequestSession(r *http.Request) {
	session, err := h.sessionFromRequest(r)
	if err != nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_ = h.store.RevokePortalSession(ctx, session.ID)
}

func (h *Handler) requestIP(r *http.Request) string {
	if h.clientIP != nil {
		return h.clientIP(r)
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func newSessionToken() (plain string, hash string, err error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", "", err
	}

	plain = base64.RawURLEncoding.EncodeToString(raw)
	return plain, store.HashToken(plain), nil
}

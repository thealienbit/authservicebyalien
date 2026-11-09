package portal

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/thealienbit/authservicebyalien/internal/auth"
	"github.com/thealienbit/authservicebyalien/internal/store"
)

func (h *Handler) LoginPage(w http.ResponseWriter, r *http.Request) {
	h.render(w, http.StatusOK, "login", Page{})
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	email := strings.ToLower(strings.TrimSpace(r.PostFormValue("email")))
	password := r.PostFormValue("password")

	page := Page{Email: email}
	if email == "" || password == "" {
		page.Error = "Email and password are required."
		h.render(w, http.StatusBadRequest, "login", page)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	admin, err := h.store.GetPlatformAdminByEmail(ctx, email)
	if errors.Is(err, store.ErrNotFound) {
		page.Error = "Invalid credentials."
		h.render(w, http.StatusUnauthorized, "login", page)
		return
	}
	if err != nil {
		h.render(w, http.StatusInternalServerError, "login", Page{Error: "Internal error."})
		return
	}

	if admin.Status != "ACTIVE" || !auth.VerifyPassword(password, admin.PasswordHash) {
		page.Error = "Invalid credentials."
		h.render(w, http.StatusUnauthorized, "login", page)
		return
	}

	h.revokeRequestSession(r)

	token, tokenHash, err := newSessionToken()
	if err != nil {
		h.render(w, http.StatusInternalServerError, "login", Page{Error: "Internal error."})
		return
	}

	csrf, _, err := newSessionToken()
	if err != nil {
		h.render(w, http.StatusInternalServerError, "login", Page{Error: "Internal error."})
		return
	}

	session, err := h.store.CreatePortalSession(
		ctx,
		admin.ID,
		tokenHash,
		csrf,
		h.requestIP(r),
		r.UserAgent(),
		sessionTTL,
	)
	if err != nil {
		h.render(w, http.StatusInternalServerError, "login", Page{Error: "Internal error."})
		return
	}

	h.setSessionCookie(w, token, session.ExpiresAt)
	http.Redirect(w, r, "/portal/", http.StatusSeeOther)
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	if session, ok := sessionFromContext(r.Context()); ok {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		_ = h.store.RevokePortalSession(ctx, session.ID)
		cancel()
	}

	h.clearSessionCookie(w)
	http.Redirect(w, r, "/portal/login", http.StatusSeeOther)
}

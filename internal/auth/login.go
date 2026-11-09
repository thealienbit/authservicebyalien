package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/thealienbit/authservicebyalien/internal/store"
)

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	app, ok := AppFromContext(r.Context())
	if !ok {
		http.Error(w, "Internal Error", http.StatusInternalServerError)
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, genericLoginError(), http.StatusUnauthorized)
		return
	}

	email := strings.TrimSpace(strings.ToLower(req.Email))
	if email == "" || req.Password == "" {
		http.Error(w, genericLoginError(), http.StatusUnauthorized)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var (
		userID        uuid.UUID
		passwordHash  string
		emailVerified bool
	)

	err := h.DB.QueryRow(ctx, `
	SELECT u.id, pc.password_hash, ue.is_verified
	FROM users u
	JOIN user_emails ue ON ue.user_id = u.id
	JOIN password_credentials pc ON pc.user_id = u.id
	WHERE ue.email = $1
	  AND ue.app_id = $2
	  AND ue.is_primary = true
	  AND u.status = 'ACTIVE'
	`, email, app.ID).Scan(&userID, &passwordHash, &emailVerified)

	if err == pgx.ErrNoRows {
		http.Error(w, genericLoginError(), http.StatusUnauthorized)
		return
	}
	if err != nil {
		http.Error(w, genericLoginError(), http.StatusUnauthorized)
		return
	}

	if !VerifyPassword(req.Password, passwordHash) {
		http.Error(w, genericLoginError(), http.StatusUnauthorized)
		return
	}

	if !emailVerified {
		http.Error(w, "Email not verified", http.StatusForbidden)
		return
	}

	membership, err := h.Store.GetMembershipByUser(ctx, userID)
	if errors.Is(err, store.ErrNotFound) {
		http.Error(w, "No organization membership", http.StatusForbidden)
		return
	}
	if err != nil {
		http.Error(w, "Internal Error", http.StatusInternalServerError)
		return
	}

	accessToken, err := h.JWTIssuer.IssueAccessToken(AccessTokenClaims{
		UserID:        userID,
		Email:         email,
		EmailVerified: emailVerified,
		AppID:         app.ID,
		OrgID:         membership.OrganizationID,
		Role:          membership.Role,
	})
	if err != nil {
		http.Error(w, "Internal Error", http.StatusInternalServerError)
		return
	}

	refreshToken, err := h.createRefreshToken(ctx, userID, r.UserAgent(), r.RemoteAddr)
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	resp := LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiredIn:    AccessTokenTTLSeconds,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func genericLoginError() string {
	return "Invalid Credentials"
}

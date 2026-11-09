package auth

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/thealienbit/authservicebyalien/internal/store"
)

func (h *Handler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	app, ok := AppFromContext(r.Context())
	if !ok {
		http.Error(w, "Internal Error", http.StatusInternalServerError)
		return
	}

	var req RefreshTokenRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid Token", http.StatusUnauthorized)
		return
	}

	if req.RefreshToken == "" {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	sum := sha256.Sum256([]byte(req.RefreshToken))
	tokenHash := base64.RawURLEncoding.EncodeToString(sum[:])

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	tx, err := h.DB.Begin(ctx)
	if err != nil {
		http.Error(w, "Internal Error", http.StatusUnauthorized)
		return
	}
	defer tx.Rollback(ctx)

	var (
		refreshID uuid.UUID
		userID    uuid.UUID
		appID     uuid.UUID
		orgID     uuid.UUID
		email     string
		verified  bool
		role      string
		expiresAt time.Time
		revokedAt *time.Time
	)

	err = tx.QueryRow(ctx, `
		SELECT rt.id, rt.user_id, rt.expires_at, rt.revoked_at,
		       ue.email, ue.is_verified, u.app_id, m.organization_id, m.role
		FROM refresh_tokens rt
		JOIN users u ON u.id = rt.user_id
		JOIN user_emails ue ON ue.user_id = rt.user_id AND ue.is_primary = true
		JOIN memberships m ON m.user_id = rt.user_id
		WHERE rt.token_hash = $1
		FOR UPDATE OF rt
	`, tokenHash).Scan(
		&refreshID,
		&userID,
		&expiresAt,
		&revokedAt,
		&email,
		&verified,
		&appID,
		&orgID,
		&role,
	)

	if err == pgx.ErrNoRows {
		http.Error(w, "Invalid Token", http.StatusUnauthorized)
		return
	}
	if err != nil {
		http.Error(w, "Internal Error", http.StatusInternalServerError)
		return
	}

	if appID != app.ID {
		http.Error(w, "Invalid Token", http.StatusUnauthorized)
		return
	}

	// Reuse of a revoked token, or an expired one: revoke the user's sessions.
	if revokedAt != nil || time.Now().After(expiresAt) {
		_, _ = tx.Exec(ctx, `
			UPDATE refresh_tokens
			SET revoked_at = now()
			WHERE user_id = $1
			  AND revoked_at IS NULL
		`, userID)
		http.Error(w, "Invalid Token", http.StatusUnauthorized)
		return
	}

	// Rotate: revoke old token.
	_, err = tx.Exec(ctx, `
		UPDATE refresh_tokens
		SET revoked_at = now()
		WHERE id = $1
	`, refreshID)
	if err != nil {
		http.Error(w, "Internal Error", http.StatusInternalServerError)
		return
	}

	newPlain, newHash, err := GenerateToken()
	if err != nil {
		http.Error(w, "Internal Error", http.StatusInternalServerError)
		return
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO refresh_tokens (
			id, user_id, token_hash, expires_at
		) VALUES ($1, $2, $3, now() + interval '30 days')
	`, uuid.New(), userID, newHash)
	if err != nil {
		http.Error(w, "Internal Error", http.StatusInternalServerError)
		return
	}

	accessToken, err := h.JWTIssuer.IssueAccessToken(AccessTokenClaims{
		UserID:        userID,
		Email:         email,
		EmailVerified: verified,
		AppID:         appID,
		OrgID:         orgID,
		Role:          store.Role(role),
	})
	if err != nil {
		http.Error(w, "Internal Error", http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(ctx); err != nil {
		http.Error(w, "Internal Error", http.StatusInternalServerError)
		return
	}

	resp := RefreshTokenResponse{
		AccessToken:  accessToken,
		RefreshToken: newPlain,
		ExpiresIn:    AccessTokenTTLSeconds,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

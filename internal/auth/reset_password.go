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
)

func (h *Handler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req ResetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	if req.Token == "" || len(req.NewPassword) < 8 {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	sum := sha256.Sum256([]byte(req.Token))
	tokenHash := base64.RawURLEncoding.EncodeToString(sum[:])

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	tx, err := h.DB.Begin(ctx)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback(ctx)

	var userID uuid.UUID
	err = tx.QueryRow(ctx, `
		SELECT user_id
		FROM password_reset_tokens
		WHERE token_hash = $1
		  AND used_at IS NULL
		  AND expires_at > now()
		FOR UPDATE
	`, tokenHash).Scan(&userID)

	if err == pgx.ErrNoRows {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	newHash, err := HashPassword(req.NewPassword)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Update password
	_, err = tx.Exec(ctx, `
		UPDATE password_credentials
		SET password_hash = $1,
		    updated_at = now()
		WHERE user_id = $2
	`, newHash, userID)

	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Mark token used
	_, _ = tx.Exec(ctx, `
		UPDATE password_reset_tokens
		SET used_at = now()
		WHERE token_hash = $1
	`, tokenHash)

	// Revoke all refresh tokens
	_, _ = tx.Exec(ctx, `
		UPDATE refresh_tokens
		SET revoked_at = now()
		WHERE user_id = $1
	`, userID)

	if err := tx.Commit(ctx); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

package auth

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
)

func (h *Handler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	app, ok := AppFromContext(r.Context())
	if !ok {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	var req VerifyEmailRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	if req.Token == "" {
		http.Error(w, "invalid request", http.StatusBadRequest)
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

	var emailID string
	err = tx.QueryRow(ctx, `
		SELECT ev.email_id
		FROM email_verification_tokens ev
		JOIN user_emails ue ON ue.id = ev.email_id
		WHERE ev.token_hash = $1
		  AND ue.app_id = $2
		  AND ev.used_at IS NULL
		  AND ev.expires_at > now()
		FOR UPDATE OF ev
	`, tokenHash, app.ID).Scan(&emailID)

	if err == pgx.ErrNoRows {
		http.Error(w, genericVerifyMessage(), http.StatusOK)
		return
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Mark email verified
	_, err = tx.Exec(ctx, `
		UPDATE user_emails
		SET is_verified = true,
		    verified_at = now()
		WHERE id = $1
	`, emailID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Mark token used
	_, err = tx.Exec(ctx, `
		UPDATE email_verification_tokens
		SET used_at = now()
		WHERE token_hash = $1
	`, tokenHash)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(ctx); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Email verified successfully",
	})
}

func genericVerifyMessage() string {
	return "Email verified successfully"
}

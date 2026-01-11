package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (h *Handler) ResendVerification(w http.ResponseWriter, r *http.Request) {
	var req ResendVerificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, genericResendMessage(), http.StatusAccepted)
		return
	}

	email := strings.TrimSpace(strings.ToLower(req.Email))
	if email == "" {
		http.Error(w, genericResendMessage(), http.StatusAccepted)
		return
	}

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
		SELECT id
		FROM user_emails
		WHERE email = $1
		  AND is_verified = false
		  AND is_primary = true
		FOR UPDATE
	`, email).Scan(&emailID)

	if err == pgx.ErrNoRows {
		// Email does not exist or already verified — do nothing
		http.Error(w, genericResendMessage(), http.StatusAccepted)
		return
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Invalidate old tokens
	_, _ = tx.Exec(ctx, `
		UPDATE email_verification_tokens
		SET used_at = now()
		WHERE email_id = $1
		  AND used_at IS NULL
	`, emailID)

	// Generate new token
	token, tokenHash, err := GenerateToken()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO email_verification_tokens (
			id, email_id, token_hash, expires_at
		) VALUES ($1, $2, $3, now() + interval '30 minutes')
	`, uuid.New(), emailID, tokenHash)

	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(ctx); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// TODO: send email
	println("RESEND EMAIL VERIFICATION TOKEN:", token)

	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte(genericResendMessage()))
}

func genericResendMessage() string {
	return "If the email exists, a verification link has been sent."
}

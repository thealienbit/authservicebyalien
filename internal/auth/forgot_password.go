package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	emailInt "github.com/thealienbit/authservicebyalien/internal/email"
)

func (h *Handler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req ForgotPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusAccepted)
		return
	}

	email := strings.TrimSpace(strings.ToLower(req.Email))
	if email == "" {
		w.WriteHeader(http.StatusAccepted)
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

	var userID uuid.UUID
	err = tx.QueryRow(ctx, `
		SELECT u.id
		FROM users u
		JOIN user_emails ue ON ue.user_id = u.id
		WHERE ue.email = $1
		  AND ue.is_primary = true
		  AND u.status = 'ACTIVE'
	`, email).Scan(&userID)

	if err == pgx.ErrNoRows {
		// Email does not exist → return generic success
		w.WriteHeader(http.StatusAccepted)
		return
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Invalidate existing reset tokens
	_, _ = tx.Exec(ctx, `
		UPDATE password_reset_tokens
		SET used_at = now()
		WHERE user_id = $1
		  AND used_at IS NULL
	`, userID)

	// Generate new token
	plain, hash, err := GenerateToken()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO password_reset_tokens (
			id, user_id, token_hash, expires_at
		) VALUES ($1, $2, $3, now() + interval '30 minutes')
	`, uuid.New(), userID, hash)

	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(ctx); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}


	link := "https://yourapp.com/verify?token=" + plain
	subject, body := emailInt.PasswordResetEmail(link)  
	_ = h.Email.Send(email, subject, body)

	w.WriteHeader(http.StatusAccepted)
}


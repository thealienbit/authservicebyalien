package auth

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thealienbit/authservicebyalien/internal/email"
	"github.com/thealienbit/authservicebyalien/internal/store"
)

type Handler struct {
	DB        *pgxpool.Pool
	JWTIssuer *JWTIssuer
	Email     email.EmailSender
	Store     *store.Store
}

func NewHandler(db *pgxpool.Pool, jwtIssuer *JWTIssuer, emailSender email.EmailSender, appStore *store.Store) *Handler {
	return &Handler{DB: db, JWTIssuer: jwtIssuer, Email: emailSender, Store: appStore}
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	app, ok := AppFromContext(r.Context())
	if !ok {
		http.Error(w, "Internal Error", http.StatusInternalServerError)
		return
	}

	var req RegisterRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid Request", http.StatusBadRequest)
		return
	}

	// Basic validation for email and password
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if req.Email == "" || len(req.Password) < 8 {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	passwordHash, err := HashPassword(req.Password)
	if err != nil {
		http.Error(w, "Internal Error", http.StatusInternalServerError)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	tx, err := h.DB.Begin(ctx)
	if err != nil {
		http.Error(w, "Internal Error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback(ctx)

	userID := uuid.New()
	emailID := uuid.New()

	_, err = tx.Exec(ctx, `
		INSERT INTO users (id, status, app_id)
		VALUES ($1, 'ACTIVE', $2)
	`, userID, app.ID)
	if err != nil {
		// when user already exists and trying to register (ask to verify email)
		http.Error(w, genericRegisterMessage(), http.StatusAccepted)
		return
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO user_emails (id, user_id, email, app_id)
		VALUES ($1, $2, $3, $4)
	`, emailID, userID, req.Email, app.ID)
	if err != nil {
		http.Error(w, genericRegisterMessage(), http.StatusAccepted)
		return
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO password_credentials (user_id, password_hash)
		VALUES ($1, $2)
	`, userID, passwordHash)
	if err != nil {
		http.Error(w, "Internal Error", http.StatusInternalServerError)
		return
	}

	token, tokenHash, err := GenerateToken()
	if err != nil {
		http.Error(w, "Internal Error", http.StatusInternalServerError)
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
		http.Error(w, "Internal Error", http.StatusInternalServerError)
		return
	}

	link := tokenLink(app.BaseURL, "/verify", token)
	subject, body := email.VerificationEmail(link)

	if err := h.Email.Send(req.Email, subject, body); err != nil {
		log.Printf("Email send failed: %v", err)
	}

	resp := RegisterResponse{
		Message: "Registration successful. Please verify your email.",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(resp)
}

func genericRegisterMessage() string {
	return "Registration successful. Please verify your email."
}

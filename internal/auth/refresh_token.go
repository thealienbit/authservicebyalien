package auth

import (
	"context"
	"log"
	"net"

	// "time"

	"github.com/google/uuid"
)

func (h *Handler) createRefreshToken(
	ctx context.Context,
	userID uuid.UUID,
	userAgent string,
	remoteAddr string,
) (string, error) {

	ip := extractIP(remoteAddr)

	plain, hash, err := GenerateToken()
	if err != nil {
		return "", err
	}

	_, err = h.DB.Exec(ctx, `
			INSERT INTO refresh_tokens (
			id, user_id, token_hash, user_agent, ip_address, expires_at
		) VALUES ($1, $2, $3, $4, $5, now() + interval '30 days')
	`, uuid.New(), userID, hash, userAgent, ip)

	if err != nil {

		log.Printf("error during inserting refresh token %v", err)
		return "", err
	}

	return plain, nil
}


func extractIP(remoteAddr string) string {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		// In case its already an IP without a port
		return remoteAddr
	}


	return host
}

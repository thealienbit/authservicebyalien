package store

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/google/uuid"
)

type PortalSession struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Email     string
	CSRF      string
	ExpiresAt time.Time
	RevokedAt *time.Time
}

func (s *Store) CreatePortalSession(ctx context.Context, userID uuid.UUID, tokenHash, csrf, ip, userAgent string, ttl time.Duration) (*PortalSession, error) {
	session := &PortalSession{
		ID:        uuid.New(),
		UserID:    userID,
		CSRF:      csrf,
		ExpiresAt: time.Now().Add(ttl),
	}

	_, err := s.pool.Exec(ctx, `
		INSERT INTO portal_sessions (
			id, user_id, token_hash, csrf_token, ip_address, user_agent, expires_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, session.ID, session.UserID, tokenHash, session.CSRF, ipOrNil(ip), userAgent, session.ExpiresAt)
	if err != nil {
		return nil, fmt.Errorf("store: create portal session: %w", err)
	}

	return session, nil
}

func (s *Store) GetPortalSession(ctx context.Context, tokenHash string) (*PortalSession, error) {
	session := &PortalSession{}

	err := s.pool.QueryRow(ctx, `
		SELECT ps.id, ps.user_id, ue.email, ps.csrf_token, ps.expires_at, ps.revoked_at
		FROM portal_sessions ps
		JOIN user_emails ue ON ue.user_id = ps.user_id
			AND ue.is_primary = true
			AND ue.app_id IS NULL
		WHERE ps.token_hash = $1
	`, tokenHash).Scan(
		&session.ID,
		&session.UserID,
		&session.Email,
		&session.CSRF,
		&session.ExpiresAt,
		&session.RevokedAt,
	)
	if err != nil {
		return nil, wrapNotFound(err)
	}

	return session, nil
}

func (s *Store) RevokePortalSession(ctx context.Context, id uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE portal_sessions
		SET revoked_at = now()
		WHERE id = $1
		  AND revoked_at IS NULL
	`, id)
	if err != nil {
		return fmt.Errorf("store: revoke portal session: %w", err)
	}

	return nil
}

func ipOrNil(value string) any {
	if net.ParseIP(value) == nil {
		return nil
	}
	return value
}

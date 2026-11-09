package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type PlatformAdmin struct {
	ID           uuid.UUID
	Email        string
	PasswordHash string
	Status       string
}

func (s *Store) GetPlatformAdminByEmail(ctx context.Context, email string) (*PlatformAdmin, error) {
	admin := &PlatformAdmin{}

	err := s.pool.QueryRow(ctx, `
		SELECT u.id, ue.email, pc.password_hash, u.status
		FROM users u
		JOIN user_emails ue ON ue.user_id = u.id
			AND ue.app_id IS NULL
			AND ue.is_primary = true
		JOIN password_credentials pc ON pc.user_id = u.id
		WHERE ue.email = $1
		  AND u.is_platform_admin = true
		  AND u.app_id IS NULL
	`, email).Scan(&admin.ID, &admin.Email, &admin.PasswordHash, &admin.Status)
	if err != nil {
		return nil, wrapNotFound(err)
	}

	return admin, nil
}

// EnsurePlatformAdmin creates a verified platform admin if the email does not
// already exist. Used by the seed command.
func (s *Store) EnsurePlatformAdmin(ctx context.Context, email, passwordHash string) (uuid.UUID, error) {
	var existing uuid.UUID
	err := s.pool.QueryRow(ctx, `
		SELECT u.id
		FROM users u
		JOIN user_emails ue ON ue.user_id = u.id
		WHERE ue.email = $1
		  AND ue.app_id IS NULL
		  AND u.is_platform_admin = true
	`, email).Scan(&existing)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, fmt.Errorf("store: find platform admin: %w", err)
	}

	userID := uuid.New()
	emailID := uuid.New()

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return uuid.Nil, fmt.Errorf("store: begin platform admin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `
		INSERT INTO users (id, status, app_id, is_platform_admin)
		VALUES ($1, 'ACTIVE', NULL, TRUE)
	`, userID); err != nil {
		return uuid.Nil, fmt.Errorf("store: create platform admin: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO user_emails (id, user_id, email, app_id, is_verified, verified_at)
		VALUES ($1, $2, $3, NULL, TRUE, now())
	`, emailID, userID, email); err != nil {
		return uuid.Nil, wrapWrite(fmt.Errorf("store: create platform admin email: %w", err))
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO password_credentials (user_id, password_hash)
		VALUES ($1, $2)
	`, userID, passwordHash); err != nil {
		return uuid.Nil, fmt.Errorf("store: create platform admin credentials: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return uuid.Nil, fmt.Errorf("store: commit platform admin: %w", err)
	}

	return userID, nil
}

type NewOrgUser struct {
	AppID          uuid.UUID
	OrganizationID uuid.UUID
	Role           Role
	Email          string
	PasswordHash   string
}

type OrgUser struct {
	ID        uuid.UUID
	Email     string
	Role      Role
	Status    string
	CreatedAt time.Time
}

// CreateUser creates a verified app user with a membership in one transaction.
func (s *Store) CreateUser(ctx context.Context, in NewOrgUser) (uuid.UUID, error) {
	if !in.Role.Valid() {
		return uuid.Nil, fmt.Errorf("store: invalid role %q", in.Role)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return uuid.Nil, fmt.Errorf("store: begin user tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var orgAppID uuid.UUID
	err = tx.QueryRow(ctx, `SELECT app_id FROM organizations WHERE id = $1`, in.OrganizationID).Scan(&orgAppID)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, ErrNotFound
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("store: load organization: %w", err)
	}
	if orgAppID != in.AppID {
		return uuid.Nil, errors.New("store: organization does not belong to the app")
	}

	userID := uuid.New()
	emailID := uuid.New()

	if _, err := tx.Exec(ctx, `
		INSERT INTO users (id, status, app_id)
		VALUES ($1, 'ACTIVE', $2)
	`, userID, in.AppID); err != nil {
		return uuid.Nil, fmt.Errorf("store: create user: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO user_emails (id, user_id, email, app_id, is_verified, verified_at)
		VALUES ($1, $2, $3, $4, TRUE, now())
	`, emailID, userID, in.Email, in.AppID); err != nil {
		return uuid.Nil, wrapWrite(fmt.Errorf("store: create user email: %w", err))
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO password_credentials (user_id, password_hash)
		VALUES ($1, $2)
	`, userID, in.PasswordHash); err != nil {
		return uuid.Nil, fmt.Errorf("store: create user credentials: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO memberships (user_id, organization_id, role)
		VALUES ($1, $2, $3)
	`, userID, in.OrganizationID, string(in.Role)); err != nil {
		return uuid.Nil, fmt.Errorf("store: create membership: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return uuid.Nil, fmt.Errorf("store: commit user: %w", err)
	}

	return userID, nil
}

func (s *Store) ListUsersByOrganization(ctx context.Context, organizationID uuid.UUID) ([]*OrgUser, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT u.id, ue.email, m.role, u.status, u.created_at
		FROM memberships m
		JOIN users u ON u.id = m.user_id
		JOIN user_emails ue ON ue.user_id = u.id AND ue.is_primary = true
		WHERE m.organization_id = $1
		ORDER BY u.created_at
	`, organizationID)
	if err != nil {
		return nil, fmt.Errorf("store: list org users: %w", err)
	}
	defer rows.Close()

	var users []*OrgUser
	for rows.Next() {
		user := &OrgUser{}
		var role string
		if err := rows.Scan(&user.ID, &user.Email, &role, &user.Status, &user.CreatedAt); err != nil {
			return nil, fmt.Errorf("store: scan org user: %w", err)
		}
		user.Role = Role(role)
		users = append(users, user)
	}

	return users, rows.Err()
}

package store

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

func (s *Store) CreateMembership(ctx context.Context, userID, organizationID uuid.UUID, role Role) (*Membership, error) {
	if !role.Valid() {
		return nil, fmt.Errorf("store: invalid role %q", role)
	}

	membership := &Membership{
		UserID:         userID,
		OrganizationID: organizationID,
		Role:           role,
	}

	err := s.pool.QueryRow(ctx, `
		INSERT INTO memberships (user_id, organization_id, role)
		VALUES ($1, $2, $3)
		RETURNING created_at, updated_at
	`, userID, organizationID, string(role)).Scan(&membership.CreatedAt, &membership.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("store: create membership: %w", err)
	}

	return membership, nil
}

// GetMembershipByUser returns the user's single membership. v1 assumes one
// organization per user; multi-org users are a later decision.
func (s *Store) GetMembershipByUser(ctx context.Context, userID uuid.UUID) (*Membership, error) {
	membership := &Membership{}
	var role string

	err := s.pool.QueryRow(ctx, `
		SELECT user_id, organization_id, role, created_at, updated_at
		FROM memberships
		WHERE user_id = $1
		ORDER BY created_at
		LIMIT 1
	`, userID).Scan(
		&membership.UserID,
		&membership.OrganizationID,
		&role,
		&membership.CreatedAt,
		&membership.UpdatedAt,
	)
	if err != nil {
		return nil, wrapNotFound(err)
	}
	membership.Role = Role(role)

	return membership, nil
}

func (s *Store) ListMembershipsByOrganization(ctx context.Context, organizationID uuid.UUID) ([]*Membership, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT user_id, organization_id, role, created_at, updated_at
		FROM memberships
		WHERE organization_id = $1
		ORDER BY created_at
	`, organizationID)
	if err != nil {
		return nil, fmt.Errorf("store: list memberships: %w", err)
	}
	defer rows.Close()

	var memberships []*Membership
	for rows.Next() {
		membership := &Membership{}
		var role string
		if err := rows.Scan(
			&membership.UserID,
			&membership.OrganizationID,
			&role,
			&membership.CreatedAt,
			&membership.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("store: scan membership: %w", err)
		}
		membership.Role = Role(role)
		memberships = append(memberships, membership)
	}

	return memberships, rows.Err()
}

package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

func (s *Store) CreateOrganization(ctx context.Context, appID uuid.UUID, name string) (*Organization, error) {
	if name == "" {
		return nil, errors.New("store: organization name is required")
	}

	org := &Organization{
		ID:     uuid.New(),
		AppID:  appID,
		Name:   name,
		Status: OrganizationStatusActive,
	}

	err := s.pool.QueryRow(ctx, `
		INSERT INTO organizations (id, app_id, name)
		VALUES ($1, $2, $3)
		RETURNING status, created_at, updated_at
	`, org.ID, org.AppID, org.Name).Scan(&org.Status, &org.CreatedAt, &org.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("store: create organization: %w", err)
	}

	return org, nil
}

func (s *Store) GetOrganizationByID(ctx context.Context, id uuid.UUID) (*Organization, error) {
	org := &Organization{}

	err := s.pool.QueryRow(ctx, `
		SELECT id, app_id, name, status, created_at, updated_at
		FROM organizations
		WHERE id = $1
	`, id).Scan(&org.ID, &org.AppID, &org.Name, &org.Status, &org.CreatedAt, &org.UpdatedAt)
	if err != nil {
		return nil, wrapNotFound(err)
	}

	return org, nil
}

func (s *Store) ListOrganizationsByApp(ctx context.Context, appID uuid.UUID) ([]*Organization, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, app_id, name, status, created_at, updated_at
		FROM organizations
		WHERE app_id = $1
		ORDER BY created_at
	`, appID)
	if err != nil {
		return nil, fmt.Errorf("store: list organizations: %w", err)
	}
	defer rows.Close()

	var orgs []*Organization
	for rows.Next() {
		org := &Organization{}
		if err := rows.Scan(&org.ID, &org.AppID, &org.Name, &org.Status, &org.CreatedAt, &org.UpdatedAt); err != nil {
			return nil, fmt.Errorf("store: scan organization: %w", err)
		}
		orgs = append(orgs, org)
	}

	return orgs, rows.Err()
}

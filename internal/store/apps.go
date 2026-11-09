package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

// CreateApp inserts an app and returns it together with the plaintext appkey.
// The plaintext is only available here; only its hash is persisted.
func (s *Store) CreateApp(ctx context.Context, name, baseURL string) (*App, string, error) {
	if name == "" {
		return nil, "", errors.New("store: app name is required")
	}
	if baseURL == "" {
		return nil, "", errors.New("store: app base URL is required")
	}

	plain, hash, err := GenerateAppKey()
	if err != nil {
		return nil, "", err
	}

	app := &App{
		ID:      uuid.New(),
		Name:    name,
		BaseURL: baseURL,
		Status:  AppStatusActive,
	}

	err = s.pool.QueryRow(ctx, `
		INSERT INTO apps (id, name, base_url, key_hash)
		VALUES ($1, $2, $3, $4)
		RETURNING created_at, updated_at
	`, app.ID, app.Name, app.BaseURL, hash).Scan(&app.CreatedAt, &app.UpdatedAt)
	if err != nil {
		return nil, "", fmt.Errorf("store: create app: %w", err)
	}

	return app, plain, nil
}

func (s *Store) GetAppByKeyHash(ctx context.Context, keyHash string) (*App, error) {
	app := &App{}

	err := s.pool.QueryRow(ctx, `
		SELECT id, name, base_url, status, created_at, updated_at
		FROM apps
		WHERE key_hash = $1
	`, keyHash).Scan(&app.ID, &app.Name, &app.BaseURL, &app.Status, &app.CreatedAt, &app.UpdatedAt)
	if err != nil {
		return nil, wrapNotFound(err)
	}

	return app, nil
}

func (s *Store) GetAppByID(ctx context.Context, id uuid.UUID) (*App, error) {
	app := &App{}

	err := s.pool.QueryRow(ctx, `
		SELECT id, name, base_url, status, created_at, updated_at
		FROM apps
		WHERE id = $1
	`, id).Scan(&app.ID, &app.Name, &app.BaseURL, &app.Status, &app.CreatedAt, &app.UpdatedAt)
	if err != nil {
		return nil, wrapNotFound(err)
	}

	return app, nil
}

func (s *Store) ListApps(ctx context.Context) ([]*App, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, name, base_url, status, created_at, updated_at
		FROM apps
		ORDER BY created_at
	`)
	if err != nil {
		return nil, fmt.Errorf("store: list apps: %w", err)
	}
	defer rows.Close()

	var apps []*App
	for rows.Next() {
		app := &App{}
		if err := rows.Scan(&app.ID, &app.Name, &app.BaseURL, &app.Status, &app.CreatedAt, &app.UpdatedAt); err != nil {
			return nil, fmt.Errorf("store: scan app: %w", err)
		}
		apps = append(apps, app)
	}

	return apps, rows.Err()
}

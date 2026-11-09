package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/thealienbit/authservicebyalien/internal/signing"
)

const signingLockKey int64 = 7462349818

// LoadSigningKeys returns every non-retired key, oldest first.
func (s *Store) LoadSigningKeys(ctx context.Context) ([]*signing.Key, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, kid, alg, is_active, private_key_encrypted
		FROM signing_keys
		WHERE retired_at IS NULL
		ORDER BY created_at
	`)
	if err != nil {
		return nil, fmt.Errorf("store: list signing keys: %w", err)
	}
	defer rows.Close()

	var keys []*signing.Key
	for rows.Next() {
		var (
			id        uuid.UUID
			kid       string
			alg       string
			active    bool
			encrypted string
		)
		if err := rows.Scan(&id, &kid, &alg, &active, &encrypted); err != nil {
			return nil, fmt.Errorf("store: scan signing key: %w", err)
		}

		key, err := s.loadSigningKey(id, kid, alg, active, encrypted)
		if err != nil {
			return nil, err
		}
		keys = append(keys, key)
	}

	return keys, rows.Err()
}

// EnsureSigningKey returns the active signing key, generating and persisting
// one if none exists. An advisory lock keeps replicas from racing.
func (s *Store) EnsureSigningKey(ctx context.Context) (*signing.Key, error) {
	keys, err := s.LoadSigningKeys(ctx)
	if err != nil {
		return nil, err
	}
	if active := signing.Active(keys); active != nil {
		return active, nil
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("store: begin signing key tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, signingLockKey); err != nil {
		return nil, fmt.Errorf("store: lock signing keys: %w", err)
	}

	var (
		id        uuid.UUID
		kid       string
		alg       string
		encrypted string
	)
	err = tx.QueryRow(ctx, `
		SELECT id, kid, alg, private_key_encrypted
		FROM signing_keys
		WHERE is_active AND retired_at IS NULL
		FOR UPDATE
	`).Scan(&id, &kid, &alg, &encrypted)
	switch {
	case err == nil:
		key, loadErr := s.loadSigningKey(id, kid, alg, true, encrypted)
		if loadErr != nil {
			return nil, loadErr
		}
		return key, nil
	case !errors.Is(err, pgx.ErrNoRows):
		return nil, fmt.Errorf("store: load active signing key: %w", err)
	}

	key, err := signing.Generate()
	if err != nil {
		return nil, err
	}

	pemBytes, err := key.MarshalPrivatePEM()
	if err != nil {
		return nil, err
	}

	encrypted, err = s.cipher.Encrypt(pemBytes)
	if err != nil {
		return nil, err
	}

	key.ID = uuid.New()
	if _, err := tx.Exec(ctx, `
		INSERT INTO signing_keys (id, kid, alg, private_key_encrypted, is_active)
		VALUES ($1, $2, $3, $4, TRUE)
	`, key.ID, key.KeyID, key.Algorithm, encrypted); err != nil {
		return nil, fmt.Errorf("store: insert signing key: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("store: commit signing key: %w", err)
	}

	return key, nil
}

func (s *Store) loadSigningKey(id uuid.UUID, kid, alg string, active bool, encrypted string) (*signing.Key, error) {
	pemBytes, err := s.cipher.Decrypt(encrypted)
	if err != nil {
		return nil, fmt.Errorf("store: decrypt signing key %s: %w", id, err)
	}

	privateKey, err := signing.ParsePrivatePEM(pemBytes)
	if err != nil {
		return nil, fmt.Errorf("store: parse signing key %s: %w", id, err)
	}

	thumbprint, err := signing.Thumbprint(&privateKey.PublicKey)
	if err != nil {
		return nil, err
	}
	if thumbprint != kid {
		return nil, fmt.Errorf("store: signing key %s kid does not match its key material", id)
	}

	return &signing.Key{
		ID:         id,
		KeyID:      kid,
		Algorithm:  alg,
		Active:     active,
		PrivateKey: privateKey,
		PublicKey:  &privateKey.PublicKey,
	}, nil
}

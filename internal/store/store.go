package store

import (
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/thealienbit/authservicebyalien/internal/signing"
)

var (
	ErrNotFound = errors.New("store: not found")
	ErrConflict = errors.New("store: conflict")
)

// Store owns persistence for tenancy data. It must not import internal/auth.
type Store struct {
	pool   *pgxpool.Pool
	cipher *signing.Cipher
}

func New(pool *pgxpool.Pool, keyCipher *signing.Cipher) *Store {
	return &Store{pool: pool, cipher: keyCipher}
}

func wrapNotFound(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

func wrapWrite(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return ErrConflict
	}
	return err
}

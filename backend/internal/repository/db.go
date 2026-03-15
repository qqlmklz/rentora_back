package repository

import (
	"context"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DB wraps pgx pool and provides migration.
type DB struct {
	Pool *pgxpool.Pool
}

// NewDB creates a connection pool and runs migrations.
func NewDB(ctx context.Context, databaseURL string) (*DB, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	log.Printf("[db] PostgreSQL connected")
	db := &DB{Pool: pool}
	if err := db.migrate(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	log.Printf("[db] migration ok: table users exists")
	return db, nil
}

// Close closes the connection pool.
func (db *DB) Close() {
	db.Pool.Close()
}

func (db *DB) migrate(ctx context.Context) error {
	_, err := db.Pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS users (
			id            SERIAL PRIMARY KEY,
			name          TEXT NOT NULL,
			email         TEXT NOT NULL UNIQUE,
			phone         TEXT,
			password_hash TEXT NOT NULL,
			avatar        TEXT,
			created_at    TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at    TIMESTAMP NOT NULL DEFAULT NOW()
		)
	`)
	if err != nil {
		return err
	}
	_, err = db.Pool.Exec(ctx, `ALTER TABLE users ADD COLUMN IF NOT EXISTS phone TEXT`)
	return err
}

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
	if err != nil {
		return err
	}
	_, err = db.Pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS favorites (
			id          SERIAL PRIMARY KEY,
			user_id     INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			property_id INT NOT NULL,
			created_at  TIMESTAMP NOT NULL DEFAULT NOW(),
			UNIQUE (user_id, property_id)
		)
	`)
	if err != nil {
		return err
	}

	_, err = db.Pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS properties (
			id            SERIAL PRIMARY KEY,
			user_id       INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			rent_type     TEXT NOT NULL,
			category      TEXT NOT NULL,
			price         INT NOT NULL,
			property_type TEXT NOT NULL,
			title         TEXT NOT NULL,
			city          TEXT NOT NULL,
			district      TEXT NOT NULL,
			utilities_included BOOLEAN NOT NULL DEFAULT FALSE,
			utilities_price    INT,
			deposit            INT,
			commission_percent INT,
			prepayment         TEXT,
			children_allowed   BOOLEAN NOT NULL DEFAULT FALSE,
			pets_allowed       BOOLEAN NOT NULL DEFAULT FALSE,
			address       TEXT NOT NULL,
			metro         TEXT,
			apartment_number TEXT,
			rooms         INT NOT NULL,
			total_area    DOUBLE PRECISION NOT NULL,
			living_area   DOUBLE PRECISION,
			kitchen_area  DOUBLE PRECISION,
			floor         INT,
			total_floors  INT,
			created_at    TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at    TIMESTAMP NOT NULL DEFAULT NOW()
		)
	`)
	if err != nil {
		return err
	}

	// Backward compatible columns for old schema (if table existed already).
	_, err = db.Pool.Exec(ctx, `
		ALTER TABLE properties
			ADD COLUMN IF NOT EXISTS user_id INT,
			ADD COLUMN IF NOT EXISTS rent_type TEXT,
			ADD COLUMN IF NOT EXISTS title TEXT,
			ADD COLUMN IF NOT EXISTS city TEXT,
			ADD COLUMN IF NOT EXISTS district TEXT,
			ADD COLUMN IF NOT EXISTS utilities_included BOOLEAN NOT NULL DEFAULT FALSE,
			ADD COLUMN IF NOT EXISTS utilities_price INT,
			ADD COLUMN IF NOT EXISTS deposit INT,
			ADD COLUMN IF NOT EXISTS commission_percent INT,
			ADD COLUMN IF NOT EXISTS prepayment TEXT,
			ADD COLUMN IF NOT EXISTS children_allowed BOOLEAN NOT NULL DEFAULT FALSE,
			ADD COLUMN IF NOT EXISTS pets_allowed BOOLEAN NOT NULL DEFAULT FALSE,
			ADD COLUMN IF NOT EXISTS address TEXT,
			ADD COLUMN IF NOT EXISTS metro TEXT,
			ADD COLUMN IF NOT EXISTS apartment_number TEXT,
			ADD COLUMN IF NOT EXISTS total_area DOUBLE PRECISION,
			ADD COLUMN IF NOT EXISTS living_area DOUBLE PRECISION,
			ADD COLUMN IF NOT EXISTS kitchen_area DOUBLE PRECISION,
			ADD COLUMN IF NOT EXISTS floor INT,
			ADD COLUMN IF NOT EXISTS total_floors INT,
			ADD COLUMN IF NOT EXISTS housing_type TEXT,
			ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP NOT NULL DEFAULT NOW()
	`)
	if err != nil {
		return err
	}

	// Fix old 'area' column if exists - make it nullable or set default
	_, _ = db.Pool.Exec(ctx, `ALTER TABLE properties ALTER COLUMN area DROP NOT NULL`)
	_, _ = db.Pool.Exec(ctx, `ALTER TABLE properties ALTER COLUMN area SET DEFAULT 0`)
	if err != nil {
		return err
	}

	_, err = db.Pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS property_images (
			id          SERIAL PRIMARY KEY,
			property_id INT NOT NULL REFERENCES properties(id) ON DELETE CASCADE,
			image_url   TEXT NOT NULL,
			created_at  TIMESTAMP NOT NULL DEFAULT NOW()
		)
	`)
	return err
}

package repository

import (
	"context"
	"errors"

	"rentora/backend/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// ErrDuplicateEmail is returned when insert violates unique constraint on email.
var ErrDuplicateEmail = errors.New("duplicate email")

// CreateUser inserts a user. Timestamps are set by the database.
// Returns ErrDuplicateEmail if email already exists.
func (db *DB) CreateUser(ctx context.Context, u *models.User) error {
	err := db.Pool.QueryRow(ctx, `
		INSERT INTO users (name, email, phone, password_hash, avatar)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at
	`, u.Name, u.Email, u.Phone, u.PasswordHash, u.Avatar).Scan(
		&u.ID, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrDuplicateEmail
		}
		return err
	}
	return nil
}

// GetUserByEmail returns a user by email or nil if not found.
func (db *DB) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	var u models.User
	err := db.Pool.QueryRow(ctx, `
		SELECT id, name, email, phone, password_hash, avatar, created_at, updated_at
		FROM users WHERE email = $1
	`, email).Scan(
		&u.ID, &u.Name, &u.Email, &u.Phone, &u.PasswordHash, &u.Avatar, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

// GetUserByID returns a user by id or nil if not found.
func (db *DB) GetUserByID(ctx context.Context, id int) (*models.User, error) {
	var u models.User
	err := db.Pool.QueryRow(ctx, `
		SELECT id, name, email, phone, password_hash, avatar, created_at, updated_at
		FROM users WHERE id = $1
	`, id).Scan(
		&u.ID, &u.Name, &u.Email, &u.Phone, &u.PasswordHash, &u.Avatar, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

// UpdateProfile updates name, email, phone and updated_at for a user.
// Returns ErrDuplicateEmail if new email is taken by another user.
func (db *DB) UpdateProfile(ctx context.Context, id int, name, email string, phone *string) error {
	_, err := db.Pool.Exec(ctx, `
		UPDATE users SET name = $1, email = $2, phone = $3, updated_at = NOW() WHERE id = $4
	`, name, email, phone, id)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrDuplicateEmail
		}
		return err
	}
	return nil
}

// UpdateAvatar sets avatar path for user.
func (db *DB) UpdateAvatar(ctx context.Context, id int, avatarPath *string) error {
	_, err := db.Pool.Exec(ctx, `UPDATE users SET avatar = $1, updated_at = NOW() WHERE id = $2`, avatarPath, id)
	return err
}

// UpdatePassword sets password_hash for user.
func (db *DB) UpdatePassword(ctx context.Context, id int, passwordHash string) error {
	_, err := db.Pool.Exec(ctx, `UPDATE users SET password_hash = $1, updated_at = NOW() WHERE id = $2`, passwordHash, id)
	return err
}

// GetUserByEmailExcludingID returns a user by email with id != excludeID (for conflict check).
func (db *DB) GetUserByEmailExcludingID(ctx context.Context, email string, excludeID int) (*models.User, error) {
	var u models.User
	err := db.Pool.QueryRow(ctx, `
		SELECT id, name, email, phone, password_hash, avatar, created_at, updated_at
		FROM users WHERE email = $1 AND id != $2
	`, email, excludeID).Scan(
		&u.ID, &u.Name, &u.Email, &u.Phone, &u.PasswordHash, &u.Avatar, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

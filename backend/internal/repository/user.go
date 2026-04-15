package repository

import (
	"context"
	"errors"

	"rentora/backend/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Ошибка, когда insert нарушает unique-ограничение по email.
var ErrDuplicateEmail = errors.New("duplicate email")

// Создаем пользователя. Временные поля выставляет сама БД.
// Если email уже есть, вернем ErrDuplicateEmail.
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

// Ищем пользователя по email; если не нашли, возвращаем nil.
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

// Ищем пользователя по id; если не нашли, возвращаем nil.
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

// Обновляем name, email, phone и updated_at у пользователя.
// Если новый email занят другим пользователем, вернем ErrDuplicateEmail.
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

// Обновляем путь к аватару пользователя.
func (db *DB) UpdateAvatar(ctx context.Context, id int, avatarPath *string) error {
	_, err := db.Pool.Exec(ctx, `UPDATE users SET avatar = $1, updated_at = NOW() WHERE id = $2`, avatarPath, id)
	return err
}

// Обновляем password_hash пользователя.
func (db *DB) UpdatePassword(ctx context.Context, id int, passwordHash string) error {
	_, err := db.Pool.Exec(ctx, `UPDATE users SET password_hash = $1, updated_at = NOW() WHERE id = $2`, passwordHash, id)
	return err
}

// Ищем пользователя по email, исключая id = excludeID (нужно для проверки конфликта).
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

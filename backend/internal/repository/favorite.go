package repository

import (
	"context"
	"errors"

	"rentora/backend/internal/models"

	"github.com/jackc/pgx/v5/pgconn"
)

// ErrFavoriteExists is returned when favorite already exists for user/property.
var ErrFavoriteExists = errors.New("favorite already exists")

// AddFavorite adds property to user's favorites.
func (db *DB) AddFavorite(ctx context.Context, userID, propertyID int) error {
	// Ensure property exists.
	var exists bool
	if err := db.Pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM properties WHERE id = $1)`, propertyID).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return ErrPropertyNotFound
	}

	_, err := db.Pool.Exec(ctx, `
		INSERT INTO favorites (user_id, property_id)
		VALUES ($1, $2)
	`, userID, propertyID)
	if err != nil {
		// Unique violation on (user_id, property_id) means already in favorites.
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrFavoriteExists
		}
		return err
	}
	return nil
}

// RemoveFavorite removes property from user's favorites.
func (db *DB) RemoveFavorite(ctx context.Context, userID, propertyID int) error {
	// Check that property exists first to be able to return 404.
	var exists bool
	if err := db.Pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM properties WHERE id = $1)`, propertyID).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return ErrPropertyNotFound
	}
	_, err := db.Pool.Exec(ctx, `
		DELETE FROM favorites WHERE user_id = $1 AND property_id = $2
	`, userID, propertyID)
	return err
}

// ListFavorites returns properties favorited by the given user.
func (db *DB) ListFavorites(ctx context.Context, userID int) ([]models.Property, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT
			p.id,
			p.title,
			p.price,
			p.property_type,
			p.rooms,
			p.total_area,
			p.city,
			p.district,
			COALESCE(
				(SELECT array_agg(pi.image_url ORDER BY pi.id)
				 FROM property_images pi
				 WHERE pi.property_id = p.id),
				'{}'
			) AS photos
		FROM favorites f
		JOIN properties p ON p.id = f.property_id
		WHERE f.user_id = $1
		ORDER BY f.created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []models.Property{} // Возвращаем [] вместо null
	for rows.Next() {
		var p models.Property
		var photos []string
		if err := rows.Scan(
			&p.ID,
			&p.Title,
			&p.Price,
			&p.PropertyType,
			&p.Rooms,
			&p.TotalArea,
			&p.City,
			&p.District,
			&photos,
		); err != nil {
			return nil, err
		}
		if photos == nil {
			photos = []string{}
		}
		p.Photos = photos
		result = append(result, p)
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}
	return result, nil
}
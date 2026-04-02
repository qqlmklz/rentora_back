package repository

import (
	"context"
)

// ListPropertyImageURLs returns all image_url values for a property (order by id).
func (db *DB) ListPropertyImageURLs(ctx context.Context, propertyID int) ([]string, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT image_url FROM property_images WHERE property_id = $1 ORDER BY id ASC
	`, propertyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var u string
		if err := rows.Scan(&u); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

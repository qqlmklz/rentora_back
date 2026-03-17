package repository

import (
	"context"
	"fmt"
	"strings"

	"rentora/backend/internal/models"
)

// PropertyFilters describes catalog filters.
type PropertyFilters struct {
	Category     string
	PropertyType string
	Rooms        int
	PriceFrom    int
	PriceTo      int
	Location     string
	Sort         string
}

// ListProperties returns properties for catalog using filters and sort.
func (db *DB) ListProperties(ctx context.Context, f PropertyFilters) ([]models.Property, error) {
	var (
		args   []interface{}
		clauses []string
	)

	// Basic filters.
	if f.Category != "" {
		args = append(args, f.Category)
		clauses = append(clauses, fmt.Sprintf("category = $%d", len(args)))
	}
	if f.PropertyType != "" {
		args = append(args, f.PropertyType)
		clauses = append(clauses, fmt.Sprintf("property_type = $%d", len(args)))
	}
	if f.Rooms > 0 {
		args = append(args, f.Rooms)
		clauses = append(clauses, fmt.Sprintf("rooms = $%d", len(args)))
	}
	if f.PriceFrom > 0 {
		args = append(args, f.PriceFrom)
		clauses = append(clauses, fmt.Sprintf("price >= $%d", len(args)))
	}
	if f.PriceTo > 0 {
		args = append(args, f.PriceTo)
		clauses = append(clauses, fmt.Sprintf("price <= $%d", len(args)))
	}
	if f.Location != "" {
		// Simple example: match city OR district by ILIKE.
		args = append(args, "%"+f.Location+"%")
		clauses = append(clauses, fmt.Sprintf("(city ILIKE $%d OR district ILIKE $%d)", len(args), len(args)))
	}

	query := `
		SELECT id, title, category, price, property_type, rooms, area, city, district, image
		FROM properties
	`
	if len(clauses) > 0 {
		query += " WHERE " + strings.Join(clauses, " AND ")
	}

	// Sorting.
	switch f.Sort {
	case "price_asc":
		query += " ORDER BY price ASC"
	case "price_desc":
		query += " ORDER BY price DESC"
	default: // newest
		query += " ORDER BY created_at DESC"
	}

	rows, err := db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var props []models.Property
	for rows.Next() {
		var p models.Property
		if err := rows.Scan(&p.ID, &p.Title, &p.Category, &p.Price, &p.PropertyType, &p.Rooms, &p.Area, &p.City, &p.District, &p.Image); err != nil {
			return nil, err
		}
		props = append(props, p)
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}
	return props, nil
}


package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"rentora/backend/internal/models"

	"github.com/jackc/pgx/v5"
)

const avatarURLPrefix = "/uploads/"

// ErrPropertyNotFound is returned when a property does not exist.
var ErrPropertyNotFound = errors.New("property not found")
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
		args    []interface{}
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
		// пока матчим по адресу, при необходимости добавим city/district
		args = append(args, "%"+f.Location+"%")
		clauses = append(clauses, fmt.Sprintf("address ILIKE $%d", len(args)))
	}
	query := `
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
		FROM properties p
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
	props := []models.Property{} // Инициализируем пустой slice, чтобы возвращать [] вместо null
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
		props = append(props, p)
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}
	return props, nil
}

// GetPropertyByID returns one property for the public detail page (no apartment_number).
func (db *DB) GetPropertyByID(ctx context.Context, id int) (*models.PropertyDetail, error) {
	row := db.Pool.QueryRow(ctx, `
		SELECT
			p.id,
			p.title,
			p.price,
			p.property_type,
			p.rooms,
			p.total_area,
			p.living_area,
			p.kitchen_area,
			p.floor,
			p.total_floors,
			p.housing_type,
			p.rent_type,
			p.address,
			p.city,
			p.district,
			p.metro,
			p.utilities_included,
			p.utilities_price,
			p.deposit,
			p.commission_percent,
			p.prepayment,
			p.children_allowed,
			p.pets_allowed,
			u.id,
			u.name,
			u.avatar,
			COALESCE(
				(SELECT array_agg(pi.image_url ORDER BY pi.id)
				 FROM property_images pi
				 WHERE pi.property_id = p.id),
				'{}'
			) AS photos
		FROM properties p
		LEFT JOIN users u ON u.id = p.user_id
		WHERE p.id = $1
	`, id)

	var d models.PropertyDetail
	var la, ka sql.NullFloat64
	var fl, tf sql.NullInt64
	var ht, metro, prep sql.NullString
	var up, dep, comm sql.NullInt64
	var ownerID sql.NullInt64
	var ownerName, ownerAvatar sql.NullString
	var photos []string

	err := row.Scan(
		&d.ID,
		&d.Title,
		&d.Price,
		&d.PropertyType,
		&d.Rooms,
		&d.TotalArea,
		&la,
		&ka,
		&fl,
		&tf,
		&ht,
		&d.RentType,
		&d.Address,
		&d.City,
		&d.District,
		&metro,
		&d.UtilitiesIncluded,
		&up,
		&dep,
		&comm,
		&prep,
		&d.ChildrenAllowed,
		&d.PetsAllowed,
		&ownerID,
		&ownerName,
		&ownerAvatar,
		&photos,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrPropertyNotFound
		}
		return nil, err
	}

	if la.Valid {
		v := la.Float64
		d.LivingArea = &v
	}
	if ka.Valid {
		v := ka.Float64
		d.KitchenArea = &v
	}
	if fl.Valid {
		v := int(fl.Int64)
		d.Floor = &v
	}
	if tf.Valid {
		v := int(tf.Int64)
		d.TotalFloors = &v
	}
	if ht.Valid {
		s := ht.String
		d.HousingType = &s
	}
	if metro.Valid {
		s := metro.String
		d.Metro = &s
	}
	if up.Valid {
		v := int(up.Int64)
		d.UtilitiesPrice = &v
	}
	if dep.Valid {
		v := int(dep.Int64)
		d.Deposit = &v
	}
	if comm.Valid {
		v := int(comm.Int64)
		d.CommissionPercent = &v
	}
	if prep.Valid {
		s := prep.String
		d.Prepayment = &s
	}
	if ownerID.Valid {
		oid := int(ownerID.Int64)
		d.OwnerID = &oid
	}
	if ownerName.Valid {
		s := ownerName.String
		d.OwnerName = &s
	}
	if ownerAvatar.Valid && ownerAvatar.String != "" {
		s := ownerAvatar.String
		if !strings.HasPrefix(s, "/") {
			s = avatarURLPrefix + strings.TrimPrefix(s, "/")
		}
		d.OwnerAvatar = &s
	}
	if photos == nil {
		photos = []string{}
	}
	d.Photos = photos

	return &d, nil
}
package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"

	"rentora/backend/internal/models"

	"github.com/jackc/pgx/v5"
)

const avatarURLPrefix = "/uploads/"

// Ошибка, когда объявление не найдено.
var ErrPropertyNotFound = errors.New("property not found")

// Ошибка, когда пользователь не владелец объявления.
var ErrPropertyForbidden = errors.New("property forbidden")
// Фильтры для каталога.
type PropertyFilters struct {
	Category     string
	PropertyType string
	RoomsExact   *int
	RoomsMin     *int
	PriceFrom    *int
	PriceTo      *int
	Location     string
	Sort         string
	CurrentUserID *int
}
// Возвращаем объявления для каталога с фильтрами и сортировкой.
func (db *DB) ListProperties(ctx context.Context, f PropertyFilters) ([]models.Property, error) {
	var (
		args    []interface{}
		clauses []string
	)
	// Базовые фильтры.
	if f.Category != "" {
		args = append(args, f.Category)
		clauses = append(clauses, fmt.Sprintf("category = $%d", len(args)))
	}
	if f.PropertyType != "" {
		args = append(args, f.PropertyType)
		clauses = append(clauses, fmt.Sprintf("property_type = $%d", len(args)))
	}
	if f.RoomsExact != nil {
		args = append(args, *f.RoomsExact)
		clauses = append(clauses, fmt.Sprintf("rooms = $%d", len(args)))
	}
	if f.RoomsMin != nil {
		args = append(args, *f.RoomsMin)
		clauses = append(clauses, fmt.Sprintf("rooms >= $%d", len(args)))
	}
	if f.PriceFrom != nil {
		args = append(args, *f.PriceFrom)
		clauses = append(clauses, fmt.Sprintf("price >= $%d", len(args)))
	}
	if f.PriceTo != nil {
		args = append(args, *f.PriceTo)
		clauses = append(clauses, fmt.Sprintf("price <= $%d", len(args)))
	}
	if f.Location != "" {
		// Ищем по нескольким полям, чтобы строка локации работала ожидаемо.
		args = append(args, "%"+f.Location+"%")
		n := len(args)
		clauses = append(clauses, fmt.Sprintf("(city ILIKE $%d OR district ILIKE $%d OR COALESCE(metro,'') ILIKE $%d OR address ILIKE $%d)", n, n, n, n))
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
			EXISTS (
				SELECT 1
				FROM contracts c
				WHERE c.property_id = p.id
				  AND c.status IN ('active', 'accepted')
			) AS is_archived,
			COALESCE(
				(SELECT array_agg(pi.image_url ORDER BY pi.id)
				 FROM property_images pi
				 WHERE pi.property_id = p.id),
				'{}'
			) AS photos
		FROM properties p
	`
	activeContractFilter := "NOT EXISTS (SELECT 1 FROM contracts c WHERE c.property_id = p.id AND c.status IN ('active', 'accepted'))"
	if f.CurrentUserID != nil {
		args = append(args, *f.CurrentUserID)
		n := len(args)
		activeContractFilter = fmt.Sprintf(`(
			NOT EXISTS (SELECT 1 FROM contracts c WHERE c.property_id = p.id AND c.status IN ('active', 'accepted'))
			OR p.user_id = $%d
			OR EXISTS (
				SELECT 1
				FROM contracts c
				WHERE c.property_id = p.id
				  AND c.status IN ('active', 'accepted')
				  AND c.tenant_id = $%d
			)
		)`, n, n)
	}
	clauses = append(clauses, activeContractFilter)
	if len(clauses) > 0 {
		query += " WHERE " + strings.Join(clauses, " AND ")
	}
	// Сортировка.
	switch f.Sort {
	case "price_asc":
		query += " ORDER BY price ASC"
	case "price_desc":
		query += " ORDER BY price DESC"
	default: // по умолчанию самые новые
		query += " ORDER BY created_at DESC"
	}
	log.Printf("[properties] catalog repo filters=%+v clauses=%v args=%v", f, clauses, args)
	log.Printf("[properties] catalog repo sql=%s", strings.Join(strings.Fields(query), " "))
	rows, err := db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	props := []models.Property{} // Инициализируем пустой slice, чтобы в JSON приходил [] вместо null.
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
			&p.IsArchived,
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

// Возвращаем одно объявление для страницы деталей (category всегда есть; apartmentNumber может скрыть handler для не-владельца).
func (db *DB) GetPropertyByID(ctx context.Context, id int) (*models.PropertyDetail, error) {
	row := db.Pool.QueryRow(ctx, `
		SELECT
			p.id,
			p.title,
			p.price,
			p.property_type,
			p.category,
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
			p.apartment_number,
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
	var apt sql.NullString
	var up, dep, comm sql.NullInt64
	var ownerID sql.NullInt64
	var ownerName, ownerAvatar sql.NullString
	var photos []string

	err := row.Scan(
		&d.ID,
		&d.Title,
		&d.Price,
		&d.PropertyType,
		&d.Category,
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
		&apt,
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
	if apt.Valid {
		s := apt.String
		d.ApartmentNumber = &s
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
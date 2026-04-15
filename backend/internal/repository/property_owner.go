package repository

import (
	"context"
	"database/sql"
	"errors"

	"rentora/backend/internal/models"

	"github.com/jackc/pgx/v5"
)

// Возвращаем карточки объявлений, которые принадлежат userID.
func (db *DB) ListPropertiesByUserID(ctx context.Context, userID int) ([]models.Property, error) {
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
		FROM properties p
		WHERE p.user_id = $1
		ORDER BY p.created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []models.Property{}
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
		out = append(out, p)
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}
	return out, nil
}

// Удаляем объявление только если оно принадлежит userID.
func (db *DB) DeletePropertyOwned(ctx context.Context, propertyID, userID int) error {
	cmd, err := db.Pool.Exec(ctx, `
		DELETE FROM properties WHERE id = $1 AND user_id = $2
	`, propertyID, userID)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() > 0 {
		return nil
	}
	var ownerID sql.NullInt64
	err = db.Pool.QueryRow(ctx, `SELECT user_id FROM properties WHERE id = $1`, propertyID).Scan(&ownerID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrPropertyNotFound
	}
	if err != nil {
		return err
	}
	if int(ownerID.Int64) != userID {
		return ErrPropertyForbidden
	}
	// Владелец тот же, но строку могли удалить параллельно.
	return ErrPropertyNotFound
}

// Загружаем полную строку объявления для merge/update. Если не нашли, вернем ErrPropertyNotFound.
func (db *DB) LoadPropertyForEdit(ctx context.Context, propertyID int) (ownerID int, in models.CreatePropertyInput, err error) {
	row := db.Pool.QueryRow(ctx, `
		SELECT
			user_id,
			rent_type, category, property_type, title, city, district, price,
			utilities_included, utilities_price, deposit, commission_percent, prepayment,
			children_allowed, pets_allowed, address, metro, apartment_number, rooms, total_area,
			living_area, kitchen_area, floor, total_floors, housing_type
		FROM properties WHERE id = $1
	`, propertyID)

	var up, dep, comm sql.NullInt64
	var prep sql.NullString
	var la, ka sql.NullFloat64
	var fl, tf sql.NullInt64
	var ht, metro, apt sql.NullString

	err = row.Scan(
		&ownerID,
		&in.RentType,
		&in.Category,
		&in.PropertyType,
		&in.Title,
		&in.City,
		&in.District,
		&in.Price,
		&in.UtilitiesIncluded,
		&up,
		&dep,
		&comm,
		&prep,
		&in.ChildrenAllowed,
		&in.PetsAllowed,
		&in.Address,
		&metro,
		&apt,
		&in.Rooms,
		&in.TotalArea,
		&la,
		&ka,
		&fl,
		&tf,
		&ht,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, in, ErrPropertyNotFound
	}
	if err != nil {
		return 0, in, err
	}

	if up.Valid {
		v := int(up.Int64)
		in.UtilitiesPrice = &v
	}
	if dep.Valid {
		v := int(dep.Int64)
		in.Deposit = &v
	}
	if comm.Valid {
		v := int(comm.Int64)
		in.CommissionPercent = &v
	}
	if prep.Valid {
		s := prep.String
		in.Prepayment = &s
	}
	if metro.Valid {
		s := metro.String
		in.Metro = &s
	}
	if apt.Valid {
		s := apt.String
		in.ApartmentNumber = &s
	}
	if la.Valid {
		v := la.Float64
		in.LivingArea = &v
	}
	if ka.Valid {
		v := ka.Float64
		in.KitchenArea = &v
	}
	if fl.Valid {
		v := int(fl.Int64)
		in.Floor = &v
	}
	if tf.Valid {
		v := int(tf.Int64)
		in.TotalFloors = &v
	}
	if ht.Valid {
		s := ht.String
		in.HousingType = &s
	}
	return ownerID, in, nil
}

// Обновляем все изменяемые колонки, если строка принадлежит userID (одним запросом).
func (db *DB) UpdatePropertyFull(ctx context.Context, propertyID, userID int, in models.CreatePropertyInput) error {
	cmd, err := db.Pool.Exec(ctx, `
		UPDATE properties SET
			rent_type = $1,
			category = $2,
			property_type = $3,
			title = $4,
			city = $5,
			district = $6,
			price = $7,
			utilities_included = $8,
			utilities_price = $9,
			deposit = $10,
			commission_percent = $11,
			prepayment = $12,
			children_allowed = $13,
			pets_allowed = $14,
			address = $15,
			metro = $16,
			apartment_number = $17,
			rooms = $18,
			total_area = $19,
			area = $20,
			living_area = $21,
			kitchen_area = $22,
			floor = $23,
			total_floors = $24,
			housing_type = $25,
			updated_at = NOW()
		WHERE id = $26 AND user_id = $27
	`,
		in.RentType,
		in.Category,
		in.PropertyType,
		in.Title,
		in.City,
		in.District,
		in.Price,
		in.UtilitiesIncluded,
		in.UtilitiesPrice,
		in.Deposit,
		in.CommissionPercent,
		in.Prepayment,
		in.ChildrenAllowed,
		in.PetsAllowed,
		in.Address,
		in.Metro,
		in.ApartmentNumber,
		in.Rooms,
		in.TotalArea,
		in.TotalArea,
		in.LivingArea,
		in.KitchenArea,
		in.Floor,
		in.TotalFloors,
		in.HousingType,
		propertyID,
		userID,
	)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return ErrPropertyNotFound
	}
	return nil
}

// В одной транзакции обновляем объявление и таблицу property_images.
func (db *DB) UpdatePropertyOwnedWithPhotos(ctx context.Context, propertyID, userID int, in models.CreatePropertyInput, deleteURLs, insertURLs []string) error {
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var owner int
	err = tx.QueryRow(ctx, `SELECT user_id FROM properties WHERE id = $1`, propertyID).Scan(&owner)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrPropertyNotFound
	}
	if err != nil {
		return err
	}
	if owner != userID {
		return ErrPropertyForbidden
	}

	_, err = tx.Exec(ctx, `
		UPDATE properties SET
			rent_type = $1,
			category = $2,
			property_type = $3,
			title = $4,
			city = $5,
			district = $6,
			price = $7,
			utilities_included = $8,
			utilities_price = $9,
			deposit = $10,
			commission_percent = $11,
			prepayment = $12,
			children_allowed = $13,
			pets_allowed = $14,
			address = $15,
			metro = $16,
			apartment_number = $17,
			rooms = $18,
			total_area = $19,
			area = $20,
			living_area = $21,
			kitchen_area = $22,
			floor = $23,
			total_floors = $24,
			housing_type = $25,
			updated_at = NOW()
		WHERE id = $26 AND user_id = $27
	`,
		in.RentType,
		in.Category,
		in.PropertyType,
		in.Title,
		in.City,
		in.District,
		in.Price,
		in.UtilitiesIncluded,
		in.UtilitiesPrice,
		in.Deposit,
		in.CommissionPercent,
		in.Prepayment,
		in.ChildrenAllowed,
		in.PetsAllowed,
		in.Address,
		in.Metro,
		in.ApartmentNumber,
		in.Rooms,
		in.TotalArea,
		in.TotalArea,
		in.LivingArea,
		in.KitchenArea,
		in.Floor,
		in.TotalFloors,
		in.HousingType,
		propertyID,
		userID,
	)
	if err != nil {
		return err
	}

	for _, u := range deleteURLs {
		if u == "" {
			continue
		}
		_, err = tx.Exec(ctx, `DELETE FROM property_images WHERE property_id = $1 AND image_url = $2`, propertyID, u)
		if err != nil {
			return err
		}
	}

	for _, u := range insertURLs {
		if u == "" {
			continue
		}
		_, err = tx.Exec(ctx, `INSERT INTO property_images (property_id, image_url) VALUES ($1, $2)`, propertyID, u)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

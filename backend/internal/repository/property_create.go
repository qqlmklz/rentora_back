package repository

import (
	"context"

	"rentora/backend/internal/models"

	"github.com/jackc/pgx/v5"
)

// CreateProperty is kept for backwards compatibility; new code should use CreatePropertyWithImages.
func (db *DB) CreateProperty(ctx context.Context, userID int, in models.CreatePropertyInput) (int, error) {
	return db.CreatePropertyWithImages(ctx, userID, in, nil)
}

// CreatePropertyWithImages creates property and images in a single transaction.
func (db *DB) CreatePropertyWithImages(ctx context.Context, userID int, in models.CreatePropertyInput, urls []string) (int, error) {
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)

	var id int
	err = tx.QueryRow(ctx, `
		INSERT INTO properties (
			user_id,
			rent_type,
			category,
			property_type,
			title,
			city,
			district,
			price,
			utilities_included,
			utilities_price,
			deposit,
			commission_percent,
			prepayment,
			children_allowed,
			pets_allowed,
			address,
			metro,
			apartment_number,
			rooms,
			total_area,
			area,
			living_area,
			kitchen_area,
			floor,
			total_floors,
			housing_type,
			created_at,
			updated_at
		)
		VALUES (
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,NOW(),NOW()
		)
		RETURNING id
	`,
		userID,
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
		in.TotalArea, // area = total_area для совместимости со старой схемой
		in.LivingArea,
		in.KitchenArea,
		in.Floor,
		in.TotalFloors,
		in.HousingType,
	).Scan(&id)
	if err != nil {
		return 0, err
	}

	if len(urls) > 0 {
		batch := &pgx.Batch{}
		for _, u := range urls {
			batch.Queue(`INSERT INTO property_images (property_id, image_url) VALUES ($1, $2)`, id, u)
		}
		br := tx.SendBatch(ctx, batch)
		for range urls {
			if _, err := br.Exec(); err != nil {
				br.Close()
				return 0, err
			}
		}
		br.Close() // Закрываем batch ДО коммита
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return id, nil
}


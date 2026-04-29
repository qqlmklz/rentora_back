package repository

import (
	"context"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Обертка над pgx pool + запуск миграций.
type DB struct {
	Pool *pgxpool.Pool
}

// Создаем пул подключений и сразу прогоняем миграции.
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

// Закрываем пул подключений.
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
		CREATE TABLE IF NOT EXISTS applications (
			id          SERIAL PRIMARY KEY,
			user_id     INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			property_id INT NOT NULL REFERENCES properties(id) ON DELETE CASCADE,
			title       TEXT NOT NULL DEFAULT '',
			category    TEXT NOT NULL DEFAULT '',
			status      TEXT NOT NULL DEFAULT 'pending',
			description TEXT NOT NULL DEFAULT '',
			priority    TEXT NOT NULL DEFAULT 'medium' CHECK (priority IN ('low', 'medium', 'high')),
			priority_status TEXT NOT NULL DEFAULT 'pending' CHECK (priority_status IN ('pending', 'ready', 'fallback')),
			priority_score DOUBLE PRECISION NOT NULL DEFAULT 0,
			priority_reason TEXT NOT NULL DEFAULT '',
			resolution_type TEXT CHECK (resolution_type IN ('owner', 'tenant')),
			request_photos JSONB NOT NULL DEFAULT '[]'::jsonb,
			expense_amount DOUBLE PRECISION,
			expense_comment TEXT,
			expense_photos JSONB NOT NULL DEFAULT '[]'::jsonb,
			expenses_submitted BOOLEAN NOT NULL DEFAULT FALSE,
			tenant_expenses_confirmed_at TIMESTAMP,
			created_at  TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at  TIMESTAMP NOT NULL DEFAULT NOW()
		)
	`)
	if err != nil {
		return err
	}
	_, err = db.Pool.Exec(ctx, `CREATE INDEX IF NOT EXISTS idx_applications_user_id ON applications(user_id)`)
	if err != nil {
		return err
	}
	_, err = db.Pool.Exec(ctx, `CREATE INDEX IF NOT EXISTS idx_applications_property_id ON applications(property_id)`)
	if err != nil {
		return err
	}
	_, err = db.Pool.Exec(ctx, `ALTER TABLE applications ADD COLUMN IF NOT EXISTS priority TEXT NOT NULL DEFAULT 'medium'`)
	if err != nil {
		return err
	}
	_, err = db.Pool.Exec(ctx, `ALTER TABLE applications ADD COLUMN IF NOT EXISTS priority_status TEXT NOT NULL DEFAULT 'pending'`)
	if err != nil {
		return err
	}
	_, err = db.Pool.Exec(ctx, `ALTER TABLE applications ADD COLUMN IF NOT EXISTS title TEXT NOT NULL DEFAULT ''`)
	if err != nil {
		return err
	}
	_, err = db.Pool.Exec(ctx, `ALTER TABLE applications ADD COLUMN IF NOT EXISTS category TEXT NOT NULL DEFAULT ''`)
	if err != nil {
		return err
	}
	_, err = db.Pool.Exec(ctx, `ALTER TABLE applications ADD COLUMN IF NOT EXISTS priority_score DOUBLE PRECISION NOT NULL DEFAULT 0`)
	if err != nil {
		return err
	}
	_, err = db.Pool.Exec(ctx, `ALTER TABLE applications ADD COLUMN IF NOT EXISTS priority_reason TEXT NOT NULL DEFAULT ''`)
	if err != nil {
		return err
	}
	_, err = db.Pool.Exec(ctx, `ALTER TABLE applications ADD COLUMN IF NOT EXISTS resolution_type TEXT`)
	if err != nil {
		return err
	}
	_, err = db.Pool.Exec(ctx, `ALTER TABLE applications ADD COLUMN IF NOT EXISTS request_photos JSONB NOT NULL DEFAULT '[]'::jsonb`)
	if err != nil {
		return err
	}
	_, err = db.Pool.Exec(ctx, `ALTER TABLE applications ADD COLUMN IF NOT EXISTS expense_amount DOUBLE PRECISION`)
	if err != nil {
		return err
	}
	_, err = db.Pool.Exec(ctx, `ALTER TABLE applications ADD COLUMN IF NOT EXISTS expense_comment TEXT`)
	if err != nil {
		return err
	}
	_, err = db.Pool.Exec(ctx, `ALTER TABLE applications ADD COLUMN IF NOT EXISTS expense_photos JSONB NOT NULL DEFAULT '[]'::jsonb`)
	if err != nil {
		return err
	}
	_, err = db.Pool.Exec(ctx, `ALTER TABLE applications ADD COLUMN IF NOT EXISTS expenses_submitted BOOLEAN NOT NULL DEFAULT FALSE`)
	if err != nil {
		return err
	}
	_, err = db.Pool.Exec(ctx, `ALTER TABLE applications ADD COLUMN IF NOT EXISTS tenant_expenses_confirmed_at TIMESTAMP`)
	if err != nil {
		return err
	}
	_, err = db.Pool.Exec(ctx, `ALTER TABLE applications ADD COLUMN IF NOT EXISTS is_archived BOOLEAN NOT NULL DEFAULT false`)
	if err != nil {
		return err
	}
	_, err = db.Pool.Exec(ctx, `UPDATE applications SET status = 'completed' WHERE status = 'resolved'`)
	if err != nil {
		return err
	}
	_, err = db.Pool.Exec(ctx, `UPDATE applications SET is_archived = true WHERE status = 'completed'`)
	if err != nil {
		return err
	}
	_, err = db.Pool.Exec(ctx, `UPDATE applications SET status = 'pending' WHERE status IS NULL OR TRIM(status) = ''`)
	if err != nil {
		return err
	}
	_, err = db.Pool.Exec(ctx, `UPDATE applications SET priority = 'medium' WHERE priority IS NULL OR TRIM(priority) = ''`)
	if err != nil {
		return err
	}
	_, err = db.Pool.Exec(ctx, `UPDATE applications SET priority_status = 'pending' WHERE priority_status IS NULL OR TRIM(priority_status) = ''`)
	if err != nil {
		return err
	}
	_, err = db.Pool.Exec(ctx, `UPDATE applications SET title = 'Без названия' WHERE TRIM(title) = ''`)
	if err != nil {
		return err
	}
	_, err = db.Pool.Exec(ctx, `UPDATE applications SET category = 'other' WHERE TRIM(category) = ''`)
	if err != nil {
		return err
	}
	_, err = db.Pool.Exec(ctx, `UPDATE applications SET priority_reason = '' WHERE priority_reason IS NULL`)
	if err != nil {
		return err
	}
	_, err = db.Pool.Exec(ctx, `UPDATE applications SET request_photos = '[]'::jsonb WHERE request_photos IS NULL`)
	if err != nil {
		return err
	}
	_, err = db.Pool.Exec(ctx, `UPDATE applications SET expense_photos = '[]'::jsonb WHERE expense_photos IS NULL`)
	if err != nil {
		return err
	}
	_, err = db.Pool.Exec(ctx, `UPDATE applications SET expenses_submitted = true WHERE expense_amount IS NOT NULL OR TRIM(COALESCE(expense_comment, '')) <> '' OR COALESCE(jsonb_array_length(expense_photos), 0) > 0`)
	if err != nil {
		return err
	}
	_, err = db.Pool.Exec(ctx, `ALTER TABLE applications ALTER COLUMN priority_score SET DEFAULT 0`)
	if err != nil {
		return err
	}
	_, err = db.Pool.Exec(ctx, `ALTER TABLE applications ALTER COLUMN priority_reason SET DEFAULT ''`)
	if err != nil {
		return err
	}
	_, err = db.Pool.Exec(ctx, `ALTER TABLE applications ALTER COLUMN request_photos SET DEFAULT '[]'::jsonb`)
	if err != nil {
		return err
	}
	_, err = db.Pool.Exec(ctx, `ALTER TABLE applications ALTER COLUMN expense_photos SET DEFAULT '[]'::jsonb`)
	if err != nil {
		return err
	}
	_, _ = db.Pool.Exec(ctx, `ALTER TABLE applications DROP CONSTRAINT IF EXISTS applications_resolution_type_check`)
	_, err = db.Pool.Exec(ctx, `
		ALTER TABLE applications ADD CONSTRAINT applications_resolution_type_check
		CHECK (resolution_type IS NULL OR resolution_type IN ('owner', 'tenant'))
	`)
	if err != nil {
		return err
	}
	_, _ = db.Pool.Exec(ctx, `ALTER TABLE applications DROP CONSTRAINT IF EXISTS applications_priority_check`)
	_, err = db.Pool.Exec(ctx, `
		ALTER TABLE applications ADD CONSTRAINT applications_priority_check
		CHECK (priority IN ('low', 'medium', 'high'))
	`)
	if err != nil {
		return err
	}
	_, _ = db.Pool.Exec(ctx, `ALTER TABLE applications DROP CONSTRAINT IF EXISTS applications_priority_status_check`)
	_, err = db.Pool.Exec(ctx, `
		ALTER TABLE applications ADD CONSTRAINT applications_priority_status_check
		CHECK (priority_status IN ('pending', 'ready', 'fallback'))
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

	// Для старой схемы: добавляем недостающие колонки, если таблица уже была создана раньше.
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

	// Для старой колонки 'area': если есть, убираем NOT NULL и ставим дефолт.
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
	if err != nil {
		return err
	}
	_, err = db.Pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS property_views (
			user_id     INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			property_id INT NOT NULL REFERENCES properties(id) ON DELETE CASCADE,
			viewed_at   TIMESTAMP NOT NULL DEFAULT NOW(),
			PRIMARY KEY (user_id, property_id)
		)
	`)
	if err != nil {
		return err
	}
	_, err = db.Pool.Exec(ctx, `CREATE INDEX IF NOT EXISTS idx_property_views_user_viewed_at ON property_views(user_id, viewed_at DESC)`)
	if err != nil {
		return err
	}

	_, err = db.Pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS chats (
			id          SERIAL PRIMARY KEY,
			property_id INT NOT NULL REFERENCES properties(id) ON DELETE CASCADE,
			seller_id   INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			buyer_id    INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			created_at  TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at  TIMESTAMP NOT NULL DEFAULT NOW(),
			CHECK (seller_id <> buyer_id),
			UNIQUE (property_id, seller_id, buyer_id)
		)
	`)
	if err != nil {
		return err
	}
	_, err = db.Pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS messages (
			id         SERIAL PRIMARY KEY,
			chat_id    INT NOT NULL REFERENCES chats(id) ON DELETE CASCADE,
			sender_id  INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			text       TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT NOW()
		)
	`)
	if err != nil {
		return err
	}
	_, err = db.Pool.Exec(ctx, `CREATE INDEX IF NOT EXISTS idx_messages_chat_id ON messages(chat_id)`)
	if err != nil {
		return err
	}
	_, err = db.Pool.Exec(ctx, `ALTER TABLE messages ADD COLUMN IF NOT EXISTS is_read BOOLEAN NOT NULL DEFAULT FALSE`)
	if err != nil {
		return err
	}
	_, err = db.Pool.Exec(ctx, `ALTER TABLE messages ADD COLUMN IF NOT EXISTS is_system BOOLEAN NOT NULL DEFAULT FALSE`)
	if err != nil {
		return err
	}
	_, err = db.Pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS contracts (
			id            SERIAL PRIMARY KEY,
			chat_id       INT NOT NULL REFERENCES chats(id) ON DELETE CASCADE,
			property_id   INT NOT NULL REFERENCES properties(id) ON DELETE CASCADE,
			landlord_id   INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			tenant_id     INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			contract_text TEXT NOT NULL,
			status        TEXT NOT NULL CHECK (status IN ('draft', 'pending', 'accepted', 'rejected', 'terminated')),
			created_at    TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at    TIMESTAMP NOT NULL DEFAULT NOW()
		)
	`)
	if err != nil {
		return err
	}
	_, err = db.Pool.Exec(ctx, `CREATE INDEX IF NOT EXISTS idx_contracts_chat ON contracts(chat_id)`)
	if err != nil {
		return err
	}
	_, err = db.Pool.Exec(ctx, `CREATE INDEX IF NOT EXISTS idx_contracts_landlord ON contracts(landlord_id)`)
	if err != nil {
		return err
	}
	_, err = db.Pool.Exec(ctx, `CREATE INDEX IF NOT EXISTS idx_contracts_tenant ON contracts(tenant_id)`)
	if err != nil {
		return err
	}
	_, err = db.Pool.Exec(ctx, `CREATE INDEX IF NOT EXISTS idx_contracts_status_accepted ON contracts(landlord_id, tenant_id, status)`)
	if err != nil {
		return err
	}
	_, err = db.Pool.Exec(ctx, `ALTER TABLE contracts ADD COLUMN IF NOT EXISTS contract_data JSONB NOT NULL DEFAULT '{}'::jsonb`)
	if err != nil {
		return err
	}
	// Обновляем CHECK по status, потому что в старых БД без 'terminated' падал UPDATE при расторжении.
	_, _ = db.Pool.Exec(ctx, `ALTER TABLE contracts DROP CONSTRAINT IF EXISTS contracts_status_check`)
	_, err = db.Pool.Exec(ctx, `
		ALTER TABLE contracts ADD CONSTRAINT contracts_status_check
		CHECK (status IN ('draft', 'pending', 'accepted', 'rejected', 'terminated'))
	`)
	if err != nil {
		return err
	}
	_, err = db.Pool.Exec(ctx, `ALTER TABLE messages ADD COLUMN IF NOT EXISTS message_type TEXT NOT NULL DEFAULT 'text'`)
	if err != nil {
		return err
	}
	_, err = db.Pool.Exec(ctx, `ALTER TABLE messages ADD COLUMN IF NOT EXISTS contract_id INT REFERENCES contracts(id) ON DELETE SET NULL`)
	if err != nil {
		return err
	}
	_, err = db.Pool.Exec(ctx, `CREATE INDEX IF NOT EXISTS idx_messages_contract_id ON messages(contract_id)`)
	if err != nil {
		return err
	}
	_, err = db.Pool.Exec(ctx, `UPDATE messages SET message_type = 'system' WHERE is_system = true AND message_type = 'text'`)
	if err != nil {
		return err
	}
	_, err = db.Pool.Exec(ctx, `ALTER TABLE messages ADD COLUMN IF NOT EXISTS contract_status TEXT`)
	if err != nil {
		return err
	}
	_, err = db.Pool.Exec(ctx, `
		UPDATE messages m
		SET contract_status = c.status
		FROM contracts c
		WHERE m.contract_id = c.id AND m.message_type = 'contract'
		  AND (m.contract_status IS NULL OR m.contract_status = '')
	`)
	if err != nil {
		return err
	}
	_, err = db.Pool.Exec(ctx, `CREATE INDEX IF NOT EXISTS idx_chats_user ON chats(seller_id, buyer_id)`)
	return err
}

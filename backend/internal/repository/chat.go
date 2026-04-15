package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

// Ошибка на случай, если пользователь вообще не участник этого чата.
var ErrChatForbidden = errors.New("chat forbidden")

// Ошибка, когда чата с таким id в базе нет.
var ErrChatNotFound = errors.New("chat not found")

// Типы сообщений для поля messages.message_type.
const (
	MessageTypeText     = "text"
	MessageTypeContract = "contract"
	MessageTypeSystem   = "system"
)

// Достаем owner id объявления из properties.user_id.
func (db *DB) GetPropertyOwnerID(ctx context.Context, propertyID int) (ownerID int, err error) {
	err = db.Pool.QueryRow(ctx, `SELECT user_id FROM properties WHERE id = $1`, propertyID).Scan(&ownerID)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrPropertyNotFound
	}
	return ownerID, err
}

// Либо создаем чат, либо берем уже существующий по связке (property_id, seller_id, buyer_id).
func (db *DB) CreateOrGetChat(ctx context.Context, propertyID, sellerID, buyerID int) (chatID int, err error) {
	err = db.Pool.QueryRow(ctx, `
		INSERT INTO chats (property_id, seller_id, buyer_id)
		VALUES ($1, $2, $3)
		ON CONFLICT (property_id, seller_id, buyer_id) DO UPDATE SET id = chats.id
		RETURNING id
	`, propertyID, sellerID, buyerID).Scan(&chatID)
	return chatID, err
}

// Возвращаем обоих участников чата: seller_id и buyer_id.
func (db *DB) GetChatParticipantIDs(ctx context.Context, chatID int) (sellerID, buyerID int, err error) {
	err = db.Pool.QueryRow(ctx, `SELECT seller_id, buyer_id FROM chats WHERE id = $1`, chatID).Scan(&sellerID, &buyerID)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, 0, ErrChatNotFound
	}
	return sellerID, buyerID, err
}

// Данные для шапки чата: объявление + собеседник.
type ChatSessionMeta struct {
	ChatID          int
	PropertyID      int
	PropertyTitle   string
	PropertyOwnerID int
	CompanionID     int
	CompanionName   string
	CompanionAvatar *string
}

// Собираем мету чата для участника. Если доступа нет (или чата нет) — вернем ErrChatForbidden.
func (db *DB) GetChatSessionMeta(ctx context.Context, chatID, userID int) (*ChatSessionMeta, error) {
	var m ChatSessionMeta
	err := db.Pool.QueryRow(ctx, `
		SELECT
			c.id,
			c.property_id,
			p.title,
			p.user_id,
			CASE WHEN c.seller_id = $2 THEN c.buyer_id ELSE c.seller_id END,
			u.name,
			NULLIF(TRIM(u.avatar), '')
		FROM chats c
		INNER JOIN properties p ON p.id = c.property_id
		INNER JOIN users u ON u.id = CASE WHEN c.seller_id = $2 THEN c.buyer_id ELSE c.seller_id END
		WHERE c.id = $1 AND (c.seller_id = $2 OR c.buyer_id = $2)
	`, chatID, userID).Scan(
		&m.ChatID,
		&m.PropertyID,
		&m.PropertyTitle,
		&m.PropertyOwnerID,
		&m.CompanionID,
		&m.CompanionName,
		&m.CompanionAvatar,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrChatForbidden
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// Проверяем, что пользователь реально участник чата (seller или buyer).
func (db *DB) IsChatParticipant(ctx context.Context, chatID, userID int) (bool, error) {
	var n int
	err := db.Pool.QueryRow(ctx, `
		SELECT 1 FROM chats WHERE id = $1 AND (seller_id = $2 OR buyer_id = $2)
	`, chatID, userID).Scan(&n)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// Одна строка списка чатов для GET /api/chats.
type ChatListRow struct {
	ChatID          int
	PropertyID      int
	PropertyOwnerID int
	PropertyTitle   string
	PropertyPhoto   *string
	CompanionID     int
	CompanionName   string
	CompanionAvatar *string
	LastMessage     *string
	LastMessageAt   *time.Time
	UnreadCount     int
}

// Возвращаем собеседника (второго участника, не текущего userID). Если доступа нет — ErrChatForbidden.
func (db *DB) GetChatCompanion(ctx context.Context, chatID, userID int) (companionID int, name string, avatar *string, err error) {
	err = db.Pool.QueryRow(ctx, `
		SELECT
			CASE WHEN c.seller_id = $2 THEN c.buyer_id ELSE c.seller_id END,
			u.name,
			NULLIF(TRIM(u.avatar), '')
		FROM chats c
		INNER JOIN users u ON u.id = CASE WHEN c.seller_id = $2 THEN c.buyer_id ELSE c.seller_id END
		WHERE c.id = $1 AND (c.seller_id = $2 OR c.buyer_id = $2)
	`, chatID, userID).Scan(&companionID, &name, &avatar)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, "", nil, ErrChatForbidden
	}
	return companionID, name, avatar, err
}

// Возвращаем все чаты пользователя (где он seller или buyer), самые свежие сверху.
func (db *DB) ListChatsForUser(ctx context.Context, userID int) ([]ChatListRow, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT
			c.id,
			c.property_id,
			p.user_id,
			p.title,
			(SELECT NULLIF(TRIM(pi.image_url), '') FROM property_images pi WHERE pi.property_id = p.id ORDER BY pi.id ASC LIMIT 1) AS prop_photo,
			CASE WHEN c.seller_id = $1 THEN c.buyer_id ELSE c.seller_id END,
			u.name,
			NULLIF(TRIM(u.avatar), ''),
			lm.text,
			lm.created_at,
			(SELECT COUNT(*)::int FROM messages m2
			 WHERE m2.chat_id = c.id AND m2.sender_id != $1 AND m2.is_read = false) AS unread_count
		FROM chats c
		INNER JOIN properties p ON p.id = c.property_id
		INNER JOIN users u ON u.id = CASE WHEN c.seller_id = $1 THEN c.buyer_id ELSE c.seller_id END
		LEFT JOIN LATERAL (
			SELECT m.text, m.created_at
			FROM messages m
			WHERE m.chat_id = c.id
			ORDER BY m.created_at DESC, m.id DESC
			LIMIT 1
		) lm ON true
		WHERE c.seller_id = $1 OR c.buyer_id = $1
		ORDER BY COALESCE(lm.created_at, c.updated_at) DESC, c.id DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ChatListRow
	for rows.Next() {
		var r ChatListRow
		if err := rows.Scan(
			&r.ChatID,
			&r.PropertyID,
			&r.PropertyOwnerID,
			&r.PropertyTitle,
			&r.PropertyPhoto,
			&r.CompanionID,
			&r.CompanionName,
			&r.CompanionAvatar,
			&r.LastMessage,
			&r.LastMessageAt,
			&r.UnreadCount,
		); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// Как хранится сообщение после чтения из базы.
type MessageRow struct {
	ID             int
	SenderID       int
	Text           string
	CreatedAt      time.Time
	IsSystem       bool
	MessageType    string
	ContractID     *int
	ContractStatus *string // статус договора подтягиваем из contracts при выдаче
}

// Отдаем сообщения чата от старых к новым (contractStatus: сначала из колонки, иначе через join к contracts).
func (db *DB) ListMessagesByChatID(ctx context.Context, chatID int) ([]MessageRow, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT m.id, m.sender_id, m.text, m.created_at, m.is_system, m.message_type, m.contract_id,
		       COALESCE(NULLIF(TRIM(m.contract_status), ''), c.status)
		FROM messages m
		LEFT JOIN contracts c ON c.id = m.contract_id
		WHERE m.chat_id = $1
		ORDER BY m.created_at ASC, m.id ASC
	`, chatID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []MessageRow
	for rows.Next() {
		var m MessageRow
		var contractID sql.NullInt64
		var contractStatus sql.NullString
		if err := rows.Scan(&m.ID, &m.SenderID, &m.Text, &m.CreatedAt, &m.IsSystem, &m.MessageType, &contractID, &contractStatus); err != nil {
			return nil, err
		}
		if contractID.Valid {
			v := int(contractID.Int64)
			m.ContractID = &v
		}
		if contractStatus.Valid {
			s := contractStatus.String
			m.ContractStatus = &s
		}
		if m.MessageType == "" {
			m.MessageType = MessageTypeText
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// Вставляем сообщение и обновляем chat.updated_at. isSystem — если это служебное сообщение (договор и т.п.).
func (db *DB) InsertMessage(ctx context.Context, chatID, senderID int, text string, isSystem bool) (MessageRow, error) {
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return MessageRow{}, err
	}
	defer tx.Rollback(ctx)

	msgType := MessageTypeText
	if isSystem {
		msgType = MessageTypeSystem
	}
	var m MessageRow
	// Вставка пройдет только если sender_id реально участник чата (это сразу проверяется на стороне БД).
	var contractIDNull sql.NullInt64
	var contractStatusNull sql.NullString
	err = tx.QueryRow(ctx, `
		INSERT INTO messages (chat_id, sender_id, text, is_read, is_system, message_type, contract_id, contract_status)
		SELECT $1, $2, $3, false, $4, $5, NULL, NULL
		FROM chats c
		WHERE c.id = $1 AND (c.seller_id = $2 OR c.buyer_id = $2)
		RETURNING id, sender_id, text, created_at, is_system, message_type, contract_id, contract_status
	`, chatID, senderID, text, isSystem, msgType).Scan(&m.ID, &m.SenderID, &m.Text, &m.CreatedAt, &m.IsSystem, &m.MessageType, &contractIDNull, &contractStatusNull)
	if errors.Is(err, pgx.ErrNoRows) {
		return MessageRow{}, ErrChatForbidden
	}
	if err != nil {
		return MessageRow{}, err
	}
	if contractIDNull.Valid {
		v := int(contractIDNull.Int64)
		m.ContractID = &v
	}
	if contractStatusNull.Valid {
		s := contractStatusNull.String
		m.ContractStatus = &s
	}

	_, err = tx.Exec(ctx, `UPDATE chats SET updated_at = NOW() WHERE id = $1`, chatID)
	if err != nil {
		return MessageRow{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return MessageRow{}, err
	}
	return m, nil
}

// Помечаем входящие как прочитанные (sender != reader), но только если reader состоит в этом чате.
func (db *DB) MarkIncomingMessagesRead(ctx context.Context, chatID, readerUserID int) error {
	_, err := db.Pool.Exec(ctx, `
		UPDATE messages m SET is_read = true
		FROM chats c
		WHERE m.chat_id = c.id AND m.chat_id = $1
		  AND (c.seller_id = $2 OR c.buyer_id = $2)
		  AND m.sender_id != $2 AND m.is_read = false
	`, chatID, readerUserID)
	return err
}

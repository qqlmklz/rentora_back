package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
)

// Статусы договора.
const (
	ContractStatusDraft      = "draft"
	ContractStatusPending    = "pending"
	ContractStatusAccepted   = "accepted"
	ContractStatusRejected   = "rejected"
	ContractStatusTerminated = "terminated"
)

// Ошибка, когда договор не найден (или апдейт не затронул ни одной строки).
var ErrContractNotFound = errors.New("contract not found")

// Ошибка, когда пользователю нельзя выполнять действие с договором.
var ErrContractForbidden = errors.New("contract forbidden")

// Берем роли и id участников чата (seller = landlord, buyer = tenant).
func (db *DB) GetChatParticipantRoles(ctx context.Context, chatID int) (propertyID, landlordID, tenantID int, err error) {
	err = db.Pool.QueryRow(ctx, `
		SELECT property_id, seller_id, buyer_id FROM chats WHERE id = $1
	`, chatID).Scan(&propertyID, &landlordID, &tenantID)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, 0, 0, ErrChatNotFound
	}
	return propertyID, landlordID, tenantID, err
}

// Полная строка договора из БД.
type ContractRow struct {
	ID            int
	ChatID        int
	PropertyID    int
	LandlordID    int
	TenantID      int
	ContractData  []byte
	ContractText  string
	Status        string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func insertMessageTx(ctx context.Context, tx pgx.Tx, chatID, senderID int, text string, isSystem bool, messageType string, contractID *int, contractStatus *string) (MessageRow, error) {
	var m MessageRow
	var contractIDNull sql.NullInt64
	var contractStatusNull sql.NullString
	err := tx.QueryRow(ctx, `
		INSERT INTO messages (chat_id, sender_id, text, is_read, is_system, message_type, contract_id, contract_status)
		SELECT $1, $2, $3, false, $4, $5, $6, $7
		FROM chats c
		WHERE c.id = $1 AND (c.seller_id = $2 OR c.buyer_id = $2)
		RETURNING id, sender_id, text, created_at, is_system, message_type, contract_id, contract_status
	`, chatID, senderID, text, isSystem, messageType, contractID, contractStatus).Scan(
		&m.ID, &m.SenderID, &m.Text, &m.CreatedAt, &m.IsSystem, &m.MessageType, &contractIDNull, &contractStatusNull,
	)
	if err != nil {
		return m, err
	}
	if contractIDNull.Valid {
		v := int(contractIDNull.Int64)
		m.ContractID = &v
	}
	if contractStatusNull.Valid {
		s := contractStatusNull.String
		m.ContractStatus = &s
	}
	return m, nil
}

// В одной транзакции создаем договор (pending) и системное сообщение.
func (db *DB) CreatePendingContractWithSystemMessage(ctx context.Context, chatID, propertyID, landlordID, tenantID int, contractDataJSON []byte, contractText, systemMsg string) (contractID int, msg MessageRow, err error) {
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return 0, MessageRow{}, err
	}
	defer tx.Rollback(ctx)

	err = tx.QueryRow(ctx, `
		INSERT INTO contracts (chat_id, property_id, landlord_id, tenant_id, contract_data, contract_text, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`, chatID, propertyID, landlordID, tenantID, contractDataJSON, contractText, ContractStatusPending).Scan(&contractID)
	if err != nil {
		return 0, MessageRow{}, err
	}

	st := ContractStatusPending
	msg, err = insertMessageTx(ctx, tx, chatID, landlordID, systemMsg, true, MessageTypeContract, &contractID, &st)
	if err != nil {
		return 0, MessageRow{}, err
	}

	_, err = tx.Exec(ctx, `UPDATE chats SET updated_at = NOW() WHERE id = $1`, chatID)
	if err != nil {
		return 0, MessageRow{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, MessageRow{}, err
	}
	return contractID, msg, nil
}

// Загружаем договор по id.
func (db *DB) GetContractByID(ctx context.Context, contractID int) (*ContractRow, error) {
	var r ContractRow
	err := db.Pool.QueryRow(ctx, `
		SELECT id, chat_id, property_id, landlord_id, tenant_id, contract_data, contract_text, status, created_at, updated_at
		FROM contracts WHERE id = $1
	`, contractID).Scan(
		&r.ID, &r.ChatID, &r.PropertyID, &r.LandlordID, &r.TenantID,
		&r.ContractData, &r.ContractText, &r.Status, &r.CreatedAt, &r.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrContractNotFound
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// В одной транзакции ставим accepted и добавляем системное сообщение.
func (db *DB) AcceptContractWithMessage(ctx context.Context, contractID, tenantID, chatID int, msgText string) (MessageRow, error) {
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return MessageRow{}, err
	}
	defer tx.Rollback(ctx)

	var _upd int
	err = tx.QueryRow(ctx, `
		UPDATE contracts SET status = $1, updated_at = NOW()
		WHERE id = $2 AND tenant_id = $3 AND status = $4
		RETURNING id
	`, ContractStatusAccepted, contractID, tenantID, ContractStatusPending).Scan(&_upd)
	if errors.Is(err, pgx.ErrNoRows) {
		return MessageRow{}, ErrContractNotFound
	}
	if err != nil {
		return MessageRow{}, err
	}

	_, err = tx.Exec(ctx, `
		UPDATE messages SET contract_status = $1
		WHERE contract_id = $2 AND message_type = $3
	`, ContractStatusAccepted, contractID, MessageTypeContract)
	if err != nil {
		return MessageRow{}, err
	}

	m, err := insertMessageTx(ctx, tx, chatID, tenantID, msgText, true, MessageTypeSystem, nil, nil)
	if err != nil {
		return MessageRow{}, err
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

// В одной транзакции ставим rejected и добавляем системное сообщение.
func (db *DB) RejectContractWithMessage(ctx context.Context, contractID, tenantID, chatID int, msgText string) (MessageRow, error) {
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return MessageRow{}, err
	}
	defer tx.Rollback(ctx)

	var _updReject int
	err = tx.QueryRow(ctx, `
		UPDATE contracts SET status = $1, updated_at = NOW()
		WHERE id = $2 AND tenant_id = $3 AND status = $4
		RETURNING id
	`, ContractStatusRejected, contractID, tenantID, ContractStatusPending).Scan(&_updReject)
	if errors.Is(err, pgx.ErrNoRows) {
		return MessageRow{}, ErrContractNotFound
	}
	if err != nil {
		return MessageRow{}, err
	}

	_, err = tx.Exec(ctx, `
		UPDATE messages SET contract_status = $1
		WHERE contract_id = $2 AND message_type = $3
	`, ContractStatusRejected, contractID, MessageTypeContract)
	if err != nil {
		return MessageRow{}, err
	}

	m, err := insertMessageTx(ctx, tx, chatID, tenantID, msgText, true, MessageTypeSystem, nil, nil)
	if err != nil {
		return MessageRow{}, err
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

// Ставим terminated и добавляем системное сообщение (вызывающий код уже проверяет, что пользователь участник).
func (db *DB) TerminateContractWithMessage(ctx context.Context, contractID, chatID, initiatorUserID int, msgText string) (MessageRow, error) {
	log.Printf("[contracts] terminate repo: contractId=%d chatId=%d initiatorUserId=%d step=begin_tx", contractID, chatID, initiatorUserID)
	if chatID < 1 {
		log.Printf("[contracts] terminate repo: contractId=%d chatId=%d reason=invalid_chat_id", contractID, chatID)
		return MessageRow{}, fmt.Errorf("invalid chat_id for contract: %d", chatID)
	}

	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		log.Printf("[contracts] terminate repo: contractId=%d step=begin_tx err=%v", contractID, err)
		return MessageRow{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var _upd int
	log.Printf("[contracts] terminate repo: contractId=%d step=update_contract_status -> %s", contractID, ContractStatusTerminated)
	err = tx.QueryRow(ctx, `
		UPDATE contracts SET status = $1, updated_at = NOW()
		WHERE id = $2 AND status = $3
		RETURNING id
	`, ContractStatusTerminated, contractID, ContractStatusAccepted).Scan(&_upd)
	if errors.Is(err, pgx.ErrNoRows) {
		log.Printf("[contracts] terminate repo: contractId=%d step=update_contract_status err=no_row (not accepted or missing)", contractID)
		return MessageRow{}, ErrContractNotFound
	}
	if err != nil {
		log.Printf("[contracts] terminate repo: contractId=%d step=update_contract_status db_err=%v", contractID, err)
		return MessageRow{}, fmt.Errorf("update contract status to terminated: %w", err)
	}

	_, err = tx.Exec(ctx, `
		UPDATE messages SET contract_status = $1
		WHERE contract_id = $2 AND message_type = $3
	`, ContractStatusTerminated, contractID, MessageTypeContract)
	if err != nil {
		log.Printf("[contracts] terminate repo: contractId=%d step=update_messages_contract_status db_err=%v", contractID, err)
		return MessageRow{}, fmt.Errorf("update messages contract_status: %w", err)
	}

	m, err := insertMessageTx(ctx, tx, chatID, initiatorUserID, msgText, true, MessageTypeSystem, nil, nil)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			log.Printf("[contracts] terminate repo: contractId=%d chatId=%d initiatorUserId=%d step=insert_system_message err=no_rows (нет чата или sender не seller/buyer)",
				contractID, chatID, initiatorUserID)
			return MessageRow{}, fmt.Errorf("system message insert returned no rows: chat_id=%d sender_id=%d (чат не найден или пользователь не участник чата): %w",
				chatID, initiatorUserID, err)
		}
		log.Printf("[contracts] terminate repo: contractId=%d chatId=%d step=insert_system_message db_err=%v", contractID, chatID, err)
		return MessageRow{}, fmt.Errorf("insert system message: %w", err)
	}
	log.Printf("[contracts] terminate repo: contractId=%d step=insert_system_message ok messageId=%d", contractID, m.ID)

	_, err = tx.Exec(ctx, `UPDATE chats SET updated_at = NOW() WHERE id = $1`, chatID)
	if err != nil {
		log.Printf("[contracts] terminate repo: contractId=%d chatId=%d step=update_chats db_err=%v", contractID, chatID, err)
		return MessageRow{}, fmt.Errorf("update chats updated_at: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		log.Printf("[contracts] terminate repo: contractId=%d step=commit err=%v", contractID, err)
		return MessageRow{}, fmt.Errorf("commit: %w", err)
	}
	log.Printf("[contracts] terminate repo: contractId=%d step=commit ok", contractID)
	return m, nil
}

// Одна строка договора для списка profile/documents.
type ContractDocumentRow struct {
	ID            int
	ChatID        int
	PropertyID    int
	PropertyTitle string
	LandlordID    int
	TenantID      int
	Status        string
	ContractData  []byte
	ContractText  string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// Возвращаем договоры accepted/terminated, где пользователь арендодатель или арендатор.
func (db *DB) ListContractsForProfileDocuments(ctx context.Context, userID int) ([]ContractDocumentRow, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT c.id, c.chat_id, c.property_id, p.title, c.landlord_id, c.tenant_id, c.status, c.contract_data, c.contract_text, c.created_at, c.updated_at
		FROM contracts c
		INNER JOIN properties p ON p.id = c.property_id
		WHERE c.status IN ($1, $2) AND (c.landlord_id = $3 OR c.tenant_id = $3)
		ORDER BY c.updated_at DESC, c.id DESC
	`, ContractStatusAccepted, ContractStatusTerminated, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ContractDocumentRow
	for rows.Next() {
		var r ContractDocumentRow
		if err := rows.Scan(&r.ID, &r.ChatID, &r.PropertyID, &r.PropertyTitle, &r.LandlordID, &r.TenantID, &r.Status, &r.ContractData, &r.ContractText, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

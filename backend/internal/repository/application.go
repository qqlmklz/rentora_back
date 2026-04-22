package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

const ApplicationStatusPending = "pending"
const ApplicationStatusInReview = "in_review"
const ApplicationStatusOwnerResolves = "owner_resolves"
const ApplicationStatusTenantResolves = "tenant_resolves"
const ApplicationStatusWaitingExpense = "waiting_expense"
const ApplicationStatusCompleted = "completed"
const ApplicationStatusResolved = "resolved"
const ApplicationPriorityPendingReason = ""

var ErrRequestNotFound = errors.New("request not found")

// ApplicationRow - одна заявка с данными объявления.
type ApplicationRow struct {
	ID               int
	UserID           int
	PropertyID       int
	Title            string
	Category         string
	Status           string
	IsArchived       bool
	Priority         string
	PriorityStatus   string
	PriorityScore    float64
	PriorityReason   string
	ResolutionType   *string
	RequestPhotos    []string
	ExpenseAmount    *float64
	ExpenseComment   *string
	ExpensePhotos    []string
	Description      string
	CreatedAt        time.Time
	UpdatedAt        time.Time
	TenantExpensesConfirmedAt sql.NullTime
	RequesterName    string
	PropertyTitle    string
	PropertyPhoto    *string
	PropertyAddress  string
	PropertyCity     string
	PropertyDistrict string
	PropertyPrice    int
	PropertyTypeName string
	PropertyRooms    int
	PropertyTotalArea float64
	PropertyPhotos   []string
}

// PropertyRequestRow - заявка по объекту владельца + данные автора заявки.
type PropertyRequestRow struct {
	ID               int
	UserID           int
	PropertyID       int
	Title            string
	Status           string
	IsArchived       bool
	Priority         string
	PriorityReason   string
	ResolutionType   *string
	RequestPhotos    []string
	ExpenseAmount    *float64
	ExpenseComment   *string
	ExpensePhotos    []string
	Description      string
	CreatedAt        time.Time
	TenantExpensesConfirmedAt sql.NullTime
	PropertyTitle    string
	PropertyPhoto    *string
	PropertyAddress  string
	PropertyCity     string
	PropertyDistrict string
	PropertyPrice    int
	PropertyTypeName string
	PropertyRooms    int
	PropertyTotalArea float64
	PropertyPhotos   []string
	RequesterID      int
	RequesterName    string
	PropertyOwnerID  int
}

type RequestDecisionRow struct {
	ID               int
	Title            string
	Description      string
	PropertyID       int
	PropertyTitle    string
	PropertyPhoto    *string
	PropertyAddress  string
	PropertyCity     string
	PropertyDistrict string
	RequesterID      int
	RequesterName    string
	PropertyOwnerID  int
	CreatedAt        time.Time
	Status           string
	IsArchived       bool
	Priority         string
	PriorityReason   string
	ResolutionType   *string
	ExpenseAmount    *float64
	ExpenseComment   *string
	ExpensePhotos    []string
	TenantExpensesConfirmedAt sql.NullTime
}

// AvailableRequestPropertyRow - объект, по которому пользователь может создать заявку.
type AvailableRequestPropertyRow struct {
	ID       int
	Title    string
	Photo    *string
	Address  string
	City     string
	District string
}

// ListApplicationsByUser возвращает заявки пользователя вместе с полями объявления.
func (db *DB) ListApplicationsByUser(ctx context.Context, userID int) ([]ApplicationRow, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT
			a.id,
			a.user_id,
			a.property_id,
			a.title,
			a.category,
			a.status,
			a.is_archived,
			a.priority,
			a.priority_status,
			a.priority_score,
			a.priority_reason,
			a.resolution_type,
			COALESCE(a.request_photos, '[]'::jsonb),
			a.expense_amount,
			a.expense_comment,
			COALESCE(a.expense_photos, '[]'::jsonb),
			a.description,
			a.created_at,
			a.updated_at,
			a.tenant_expenses_confirmed_at,
			u.name,
			p.title,
			(SELECT NULLIF(TRIM(pi.image_url), '') FROM property_images pi WHERE pi.property_id = p.id ORDER BY pi.id ASC LIMIT 1) AS property_photo,
			p.address,
			p.city,
			p.district,
			COALESCE(p.price, 0),
			COALESCE(p.property_type, ''),
			COALESCE(p.rooms, 0),
			COALESCE(p.total_area, 0),
			COALESCE(
				(SELECT array_agg(pi.image_url ORDER BY pi.id)
				 FROM property_images pi
				 WHERE pi.property_id = p.id),
				'{}'
			) AS property_photos
		FROM applications a
		INNER JOIN users u ON u.id = a.user_id
		INNER JOIN properties p ON p.id = a.property_id
		WHERE a.user_id = $1
		ORDER BY a.created_at DESC, a.id DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]ApplicationRow, 0)
	for rows.Next() {
		var r ApplicationRow
		var photo sql.NullString
		var resolutionType sql.NullString
		var requestPhotosRaw []byte
		var expenseAmount sql.NullFloat64
		var expenseComment sql.NullString
		var expensePhotosRaw []byte
		var confirmedAt sql.NullTime
		if err := rows.Scan(
			&r.ID,
			&r.UserID,
			&r.PropertyID,
			&r.Title,
			&r.Category,
			&r.Status,
			&r.IsArchived,
			&r.Priority,
			&r.PriorityStatus,
			&r.PriorityScore,
			&r.PriorityReason,
			&resolutionType,
			&requestPhotosRaw,
			&expenseAmount,
			&expenseComment,
			&expensePhotosRaw,
			&r.Description,
			&r.CreatedAt,
			&r.UpdatedAt,
			&confirmedAt,
			&r.RequesterName,
			&r.PropertyTitle,
			&photo,
			&r.PropertyAddress,
			&r.PropertyCity,
			&r.PropertyDistrict,
			&r.PropertyPrice,
			&r.PropertyTypeName,
			&r.PropertyRooms,
			&r.PropertyTotalArea,
			&r.PropertyPhotos,
		); err != nil {
			return nil, err
		}
		if photo.Valid {
			s := strings.TrimSpace(photo.String)
			if s != "" {
				r.PropertyPhoto = &s
			}
		}
		if resolutionType.Valid {
			s := strings.TrimSpace(resolutionType.String)
			if s != "" {
				r.ResolutionType = &s
			}
		}
		r.RequestPhotos = decodeStringJSONArray(requestPhotosRaw)
		if expenseAmount.Valid {
			v := expenseAmount.Float64
			r.ExpenseAmount = &v
		}
		if expenseComment.Valid {
			s := strings.TrimSpace(expenseComment.String)
			r.ExpenseComment = &s
		}
		r.ExpensePhotos = decodeStringJSONArray(expensePhotosRaw)
		r.TenantExpensesConfirmedAt = confirmedAt
		if r.PropertyPhotos == nil {
			r.PropertyPhotos = []string{}
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ListApplicationsForOwnerProperties возвращает заявки по объявлениям текущего владельца.
func (db *DB) ListApplicationsForOwnerProperties(ctx context.Context, ownerUserID int) ([]int, []PropertyRequestRow, error) {
	propertyRows, err := db.Pool.Query(ctx, `SELECT id FROM properties WHERE user_id = $1 ORDER BY id`, ownerUserID)
	if err != nil {
		return nil, nil, err
	}
	propertyIDs := make([]int, 0)
	for propertyRows.Next() {
		var id int
		if err := propertyRows.Scan(&id); err != nil {
			propertyRows.Close()
			return nil, nil, err
		}
		propertyIDs = append(propertyIDs, id)
	}
	propertyRows.Close()
	if err := propertyRows.Err(); err != nil {
		return nil, nil, err
	}
	if len(propertyIDs) == 0 {
		return propertyIDs, []PropertyRequestRow{}, nil
	}

	rows, err := db.Pool.Query(ctx, `
		SELECT
			a.id,
			a.user_id,
			a.property_id,
			a.title,
			a.status,
			a.is_archived,
			a.priority,
			a.priority_reason,
			a.resolution_type,
			COALESCE(a.request_photos, '[]'::jsonb),
			a.expense_amount,
			a.expense_comment,
			COALESCE(a.expense_photos, '[]'::jsonb),
			a.description,
			a.created_at,
			a.tenant_expenses_confirmed_at,
			p.title,
			(SELECT NULLIF(TRIM(pi.image_url), '') FROM property_images pi WHERE pi.property_id = p.id ORDER BY pi.id ASC LIMIT 1) AS property_photo,
			p.address,
			p.city,
			p.district,
			COALESCE(p.price, 0),
			COALESCE(p.property_type, ''),
			COALESCE(p.rooms, 0),
			COALESCE(p.total_area, 0),
			COALESCE(
				(SELECT array_agg(pi.image_url ORDER BY pi.id)
				 FROM property_images pi
				 WHERE pi.property_id = p.id),
				'{}'
			) AS property_photos,
			p.user_id,
			u.id,
			u.name
		FROM applications a
		INNER JOIN properties p ON p.id = a.property_id
		INNER JOIN users u ON u.id = a.user_id
		WHERE p.user_id = $1
		ORDER BY a.created_at DESC, a.id DESC
	`, ownerUserID)
	if err != nil {
		return propertyIDs, nil, err
	}
	defer rows.Close()

	out := make([]PropertyRequestRow, 0)
	for rows.Next() {
		var r PropertyRequestRow
		var photo sql.NullString
		var resolutionType sql.NullString
		var requestPhotosRaw []byte
		var expenseAmount sql.NullFloat64
		var expenseComment sql.NullString
		var expensePhotosRaw []byte
		var confirmedAt sql.NullTime
		if err := rows.Scan(
			&r.ID,
			&r.UserID,
			&r.PropertyID,
			&r.Title,
			&r.Status,
			&r.IsArchived,
			&r.Priority,
			&r.PriorityReason,
			&resolutionType,
			&requestPhotosRaw,
			&expenseAmount,
			&expenseComment,
			&expensePhotosRaw,
			&r.Description,
			&r.CreatedAt,
			&confirmedAt,
			&r.PropertyTitle,
			&photo,
			&r.PropertyAddress,
			&r.PropertyCity,
			&r.PropertyDistrict,
			&r.PropertyPrice,
			&r.PropertyTypeName,
			&r.PropertyRooms,
			&r.PropertyTotalArea,
			&r.PropertyPhotos,
			&r.PropertyOwnerID,
			&r.RequesterID,
			&r.RequesterName,
		); err != nil {
			return propertyIDs, nil, err
		}
		if photo.Valid {
			s := strings.TrimSpace(photo.String)
			if s != "" {
				r.PropertyPhoto = &s
			}
		}
		if resolutionType.Valid {
			s := strings.TrimSpace(resolutionType.String)
			if s != "" {
				r.ResolutionType = &s
			}
		}
		r.RequestPhotos = decodeStringJSONArray(requestPhotosRaw)
		if expenseAmount.Valid {
			v := expenseAmount.Float64
			r.ExpenseAmount = &v
		}
		if expenseComment.Valid {
			s := strings.TrimSpace(expenseComment.String)
			r.ExpenseComment = &s
		}
		r.ExpensePhotos = decodeStringJSONArray(expensePhotosRaw)
		r.TenantExpensesConfirmedAt = confirmedAt
		if r.PropertyPhotos == nil {
			r.PropertyPhotos = []string{}
		}
		out = append(out, r)
	}
	return propertyIDs, out, rows.Err()
}

// GetPropertyRequestForOwner возвращает одну заявку по id, если объект принадлежит владельцу.
func (db *DB) GetPropertyRequestForOwner(ctx context.Context, ownerUserID, requestID int) (*PropertyRequestRow, error) {
	var r PropertyRequestRow
	var photo sql.NullString
	var resolutionType sql.NullString
	var requestPhotosRaw []byte
	var expenseAmount sql.NullFloat64
	var expenseComment sql.NullString
	var expensePhotosRaw []byte
	var confirmedAt sql.NullTime
	err := db.Pool.QueryRow(ctx, `
		SELECT
			a.id,
			a.user_id,
			a.property_id,
			a.title,
			a.status,
			a.is_archived,
			a.priority,
			a.priority_reason,
			a.resolution_type,
			COALESCE(a.request_photos, '[]'::jsonb),
			a.expense_amount,
			a.expense_comment,
			COALESCE(a.expense_photos, '[]'::jsonb),
			a.description,
			a.created_at,
			a.tenant_expenses_confirmed_at,
			p.title,
			(SELECT NULLIF(TRIM(pi.image_url), '') FROM property_images pi WHERE pi.property_id = p.id ORDER BY pi.id ASC LIMIT 1) AS property_photo,
			p.address,
			p.city,
			p.district,
			COALESCE(p.price, 0),
			COALESCE(p.property_type, ''),
			COALESCE(p.rooms, 0),
			COALESCE(p.total_area, 0),
			COALESCE(
				(SELECT array_agg(pi.image_url ORDER BY pi.id)
				 FROM property_images pi
				 WHERE pi.property_id = p.id),
				'{}'
			) AS property_photos,
			p.user_id,
			u.id,
			u.name
		FROM applications a
		INNER JOIN properties p ON p.id = a.property_id
		INNER JOIN users u ON u.id = a.user_id
		WHERE a.id = $1 AND p.user_id = $2
	`, requestID, ownerUserID).Scan(
		&r.ID,
		&r.UserID,
		&r.PropertyID,
		&r.Title,
		&r.Status,
		&r.IsArchived,
		&r.Priority,
		&r.PriorityReason,
		&resolutionType,
		&requestPhotosRaw,
		&expenseAmount,
		&expenseComment,
		&expensePhotosRaw,
		&r.Description,
		&r.CreatedAt,
		&confirmedAt,
		&r.PropertyTitle,
		&photo,
		&r.PropertyAddress,
		&r.PropertyCity,
		&r.PropertyDistrict,
		&r.PropertyPrice,
		&r.PropertyTypeName,
		&r.PropertyRooms,
		&r.PropertyTotalArea,
		&r.PropertyPhotos,
		&r.PropertyOwnerID,
		&r.RequesterID,
		&r.RequesterName,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrRequestNotFound
	}
	if err != nil {
		return nil, err
	}
	if photo.Valid {
		s := strings.TrimSpace(photo.String)
		if s != "" {
			r.PropertyPhoto = &s
		}
	}
	if resolutionType.Valid {
		s := strings.TrimSpace(resolutionType.String)
		if s != "" {
			r.ResolutionType = &s
		}
	}
	r.RequestPhotos = decodeStringJSONArray(requestPhotosRaw)
	if expenseAmount.Valid {
		v := expenseAmount.Float64
		r.ExpenseAmount = &v
	}
	if expenseComment.Valid {
		s := strings.TrimSpace(expenseComment.String)
		r.ExpenseComment = &s
	}
	r.ExpensePhotos = decodeStringJSONArray(expensePhotosRaw)
	r.TenantExpensesConfirmedAt = confirmedAt
	if r.PropertyPhotos == nil {
		r.PropertyPhotos = []string{}
	}
	return &r, nil
}

// CreateApplication создает новую заявку и сразу отдает сохраненную строку с данными объявления.
func (db *DB) CreateApplication(ctx context.Context, userID, propertyID int, title, description, category string, requestPhotos []string) (*ApplicationRow, error) {
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var id int
	var createdAt time.Time
	requestPhotosJSON, err := json.Marshal(requestPhotos)
	if err != nil {
		return nil, err
	}
	// status: явно pending — новые заявки должны попадать в activeRequests (не completed).
	err = tx.QueryRow(ctx, `
		INSERT INTO applications (user_id, property_id, title, description, category, status, priority, priority_status, priority_score, priority_reason, request_photos)
		SELECT $1, p.id, $3, $4, $5, $6, 'medium', 'pending', 0, $7, $8::jsonb
		FROM properties p
		WHERE p.id = $2
		RETURNING id, created_at
	`, userID, propertyID, title, description, category, ApplicationStatusPending, ApplicationPriorityPendingReason, string(requestPhotosJSON)).Scan(&id, &createdAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrPropertyNotFound
	}
	if err != nil {
		return nil, err
	}

	var row ApplicationRow
	var photo sql.NullString
	var resolutionType sql.NullString
	var requestPhotosRaw []byte
	var expenseAmount sql.NullFloat64
	var expenseComment sql.NullString
	var expensePhotosRaw []byte
	var confirmedAtCreate sql.NullTime
	err = tx.QueryRow(ctx, `
		SELECT
			a.id,
			a.user_id,
			a.property_id,
			a.title,
			a.category,
			a.status,
			a.is_archived,
			a.priority,
			a.priority_status,
			a.priority_score,
			a.priority_reason,
			a.resolution_type,
			COALESCE(a.request_photos, '[]'::jsonb),
			a.expense_amount,
			a.expense_comment,
			COALESCE(a.expense_photos, '[]'::jsonb),
			a.description,
			a.created_at,
			a.updated_at,
			a.tenant_expenses_confirmed_at,
			p.title,
			(SELECT NULLIF(TRIM(pi.image_url), '') FROM property_images pi WHERE pi.property_id = p.id ORDER BY pi.id ASC LIMIT 1) AS property_photo,
			p.address,
			p.city,
			p.district,
			COALESCE(p.price, 0),
			COALESCE(p.property_type, ''),
			COALESCE(p.rooms, 0),
			COALESCE(p.total_area, 0),
			COALESCE(
				(SELECT array_agg(pi.image_url ORDER BY pi.id)
				 FROM property_images pi
				 WHERE pi.property_id = p.id),
				'{}'
			) AS property_photos
		FROM applications a
		INNER JOIN properties p ON p.id = a.property_id
		WHERE a.id = $1
	`, id).Scan(
		&row.ID,
		&row.UserID,
		&row.PropertyID,
		&row.Title,
		&row.Category,
		&row.Status,
		&row.IsArchived,
		&row.Priority,
		&row.PriorityStatus,
		&row.PriorityScore,
		&row.PriorityReason,
		&resolutionType,
		&requestPhotosRaw,
		&expenseAmount,
		&expenseComment,
		&expensePhotosRaw,
		&row.Description,
		&row.CreatedAt,
		&row.UpdatedAt,
		&confirmedAtCreate,
		&row.PropertyTitle,
		&photo,
		&row.PropertyAddress,
		&row.PropertyCity,
		&row.PropertyDistrict,
		&row.PropertyPrice,
		&row.PropertyTypeName,
		&row.PropertyRooms,
		&row.PropertyTotalArea,
		&row.PropertyPhotos,
	)
	if err != nil {
		return nil, err
	}
	row.TenantExpensesConfirmedAt = confirmedAtCreate
	if photo.Valid {
		s := strings.TrimSpace(photo.String)
		if s != "" {
			row.PropertyPhoto = &s
		}
	}
	if resolutionType.Valid {
		s := strings.TrimSpace(resolutionType.String)
		if s != "" {
			row.ResolutionType = &s
		}
	}
	row.RequestPhotos = decodeStringJSONArray(requestPhotosRaw)
	if expenseAmount.Valid {
		v := expenseAmount.Float64
		row.ExpenseAmount = &v
	}
	if expenseComment.Valid {
		s := strings.TrimSpace(expenseComment.String)
		row.ExpenseComment = &s
	}
	row.ExpensePhotos = decodeStringJSONArray(expensePhotosRaw)
	if row.PropertyPhotos == nil {
		row.PropertyPhotos = []string{}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &row, nil
}

func (db *DB) GetRequestDecisionInfo(ctx context.Context, requestID int) (*RequestDecisionRow, error) {
	var row RequestDecisionRow
	var photo sql.NullString
	var resolutionType sql.NullString
	var expenseAmount sql.NullFloat64
	var expenseComment sql.NullString
	var expensePhotosRaw []byte
	var confirmedAt sql.NullTime
	err := db.Pool.QueryRow(ctx, `
		SELECT
			a.id,
			a.title,
			a.description,
			a.property_id,
			p.title,
			(SELECT NULLIF(TRIM(pi.image_url), '') FROM property_images pi WHERE pi.property_id = p.id ORDER BY pi.id ASC LIMIT 1) AS property_photo,
			p.address,
			p.city,
			p.district,
			a.user_id,
			u.name,
			p.user_id,
			a.created_at,
			a.status,
			a.is_archived,
			a.priority,
			a.priority_reason,
			a.resolution_type,
			a.expense_amount,
			a.expense_comment,
			COALESCE(a.expense_photos, '[]'::jsonb),
			a.tenant_expenses_confirmed_at
		FROM applications a
		INNER JOIN properties p ON p.id = a.property_id
		INNER JOIN users u ON u.id = a.user_id
		WHERE a.id = $1
	`, requestID).Scan(
		&row.ID,
		&row.Title,
		&row.Description,
		&row.PropertyID,
		&row.PropertyTitle,
		&photo,
		&row.PropertyAddress,
		&row.PropertyCity,
		&row.PropertyDistrict,
		&row.RequesterID,
		&row.RequesterName,
		&row.PropertyOwnerID,
		&row.CreatedAt,
		&row.Status,
		&row.IsArchived,
		&row.Priority,
		&row.PriorityReason,
		&resolutionType,
		&expenseAmount,
		&expenseComment,
		&expensePhotosRaw,
		&confirmedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrRequestNotFound
	}
	if err != nil {
		return nil, err
	}
	if resolutionType.Valid {
		s := strings.TrimSpace(resolutionType.String)
		if s != "" {
			row.ResolutionType = &s
		}
	}
	if photo.Valid {
		s := strings.TrimSpace(photo.String)
		if s != "" {
			row.PropertyPhoto = &s
		}
	}
	if expenseAmount.Valid {
		v := expenseAmount.Float64
		row.ExpenseAmount = &v
	}
	if expenseComment.Valid {
		s := strings.TrimSpace(expenseComment.String)
		row.ExpenseComment = &s
	}
	row.ExpensePhotos = decodeStringJSONArray(expensePhotosRaw)
	row.TenantExpensesConfirmedAt = confirmedAt
	return &row, nil
}

func (db *DB) ApplyRequestDecision(ctx context.Context, requestID int, status, resolutionType string) error {
	sqlQuery := `
		UPDATE applications
		SET status = $1,
		    resolution_type = $2,
		    updated_at = NOW()
		WHERE id = $3
	`
	_, err := db.Pool.Exec(ctx, sqlQuery, status, resolutionType, requestID)
	if err != nil {
		return fmt.Errorf("update request decision failed requestID=%d query=%q: %w", requestID, strings.TrimSpace(sqlQuery), err)
	}
	return nil
}

func (db *DB) ApplyRequestExpense(ctx context.Context, requestID int, expenseAmount float64, expenseComment string, expensePhotos []string, nextStatus string) error {
	photosJSON, err := json.Marshal(expensePhotos)
	if err != nil {
		return fmt.Errorf("marshal request expense photos failed requestID=%d: %w", requestID, err)
	}
	sqlQuery := `
		UPDATE applications
		SET expense_amount = $1,
		    expense_comment = $2,
		    expense_photos = $3::jsonb,
		    status = $4,
		    updated_at = NOW()
		WHERE id = $5
	`
	_, err = db.Pool.Exec(ctx, sqlQuery, expenseAmount, expenseComment, string(photosJSON), nextStatus, requestID)
	if err != nil {
		return fmt.Errorf("update request expense failed requestID=%d query=%q: %w", requestID, strings.TrimSpace(sqlQuery), err)
	}
	return nil
}

// ConfirmTenantExpensesResult — строка из БД сразу после UPDATE … RETURNING.
type ConfirmTenantExpensesResult struct {
	ID         int
	Status     string
	IsArchived bool
}

// ConfirmTenantExpenses помечает расходы жильца как подтверждённые владельцем.
// В БД выставляются status = completed и is_archived = true; RETURNING возвращает сохранённые значения строки.
func (db *DB) ConfirmTenantExpenses(ctx context.Context, requestID int) (*ConfirmTenantExpensesResult, error) {
	sqlQuery := `
		UPDATE applications
		SET status = $1,
		    is_archived = TRUE,
		    tenant_expenses_confirmed_at = NOW(),
		    updated_at = NOW()
		WHERE id = $2
		  AND resolution_type = 'tenant'
		  AND expense_amount IS NOT NULL
		  AND TRIM(COALESCE(expense_comment, '')) <> ''
		  AND status = $3
		  AND tenant_expenses_confirmed_at IS NULL
		RETURNING id, status, is_archived
	`
	var id int
	var status string
	var isArchived bool
	err := db.Pool.QueryRow(ctx, sqlQuery, ApplicationStatusCompleted, requestID, ApplicationStatusWaitingExpense).Scan(&id, &status, &isArchived)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("confirm tenant expenses failed requestID=%d query=%q: %w", requestID, strings.TrimSpace(sqlQuery), err)
	}
	return &ConfirmTenantExpensesResult{ID: id, Status: status, IsArchived: isArchived}, nil
}

// ResolveOwnerRequest завершает сценарий «устраняет владелец»: owner_resolves → resolved.
func (db *DB) ResolveOwnerRequest(ctx context.Context, requestID, ownerUserID int) (bool, error) {
	sqlQuery := `
		UPDATE applications a
		SET status = $1,
		    is_archived = TRUE,
		    updated_at = NOW()
		FROM properties p
		WHERE a.id = $2
		  AND a.property_id = p.id
		  AND p.user_id = $3
		  AND a.resolution_type = 'owner'
		  AND a.status = $4
	`
	cmd, err := db.Pool.Exec(ctx, sqlQuery, ApplicationStatusResolved, requestID, ownerUserID, ApplicationStatusOwnerResolves)
	if err != nil {
		return false, fmt.Errorf("resolve owner request failed requestID=%d query=%q: %w", requestID, strings.TrimSpace(sqlQuery), err)
	}
	return cmd.RowsAffected() > 0, nil
}

// CompleteOwnerFlowResult — строка после owner flow → completed (RETURNING).
type CompleteOwnerFlowResult struct {
	ID         int
	Status     string
	IsArchived bool
}

// CompleteOwnerRequestToCompleted: owner_resolves → completed, только владелец и resolution_type = owner.
func (db *DB) CompleteOwnerRequestToCompleted(ctx context.Context, requestID, ownerUserID int) (*CompleteOwnerFlowResult, error) {
	sqlQuery := `
		UPDATE applications a
		SET status = $1,
		    is_archived = TRUE,
		    updated_at = NOW()
		FROM properties p
		WHERE a.id = $2
		  AND a.property_id = p.id
		  AND p.user_id = $3
		  AND a.resolution_type = 'owner'
		  AND a.status = $4
		RETURNING a.id, a.status, a.is_archived
	`
	var id int
	var status string
	var isArchived bool
	err := db.Pool.QueryRow(ctx, sqlQuery, ApplicationStatusCompleted, requestID, ownerUserID, ApplicationStatusOwnerResolves).Scan(&id, &status, &isArchived)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("complete owner request to completed failed requestID=%d query=%q: %w", requestID, strings.TrimSpace(sqlQuery), err)
	}
	return &CompleteOwnerFlowResult{ID: id, Status: status, IsArchived: isArchived}, nil
}

// UpdateApplicationPriority обновляет AI-поля приоритета у уже созданной заявки.
func (db *DB) UpdateApplicationPriority(ctx context.Context, requestID int, priority string, priorityStatus string, priorityScore float64, priorityReason string) error {
	sqlQuery := `
		UPDATE applications
		SET priority = $1,
		    priority_status = $2,
		    priority_score = $3,
		    priority_reason = $4,
		    updated_at = NOW()
		WHERE id = $5
	`
	_, err := db.Pool.Exec(ctx, sqlQuery, priority, priorityStatus, priorityScore, priorityReason, requestID)
	if err != nil {
		return fmt.Errorf("update applications priority failed requestID=%d query=%q: %w", requestID, strings.TrimSpace(sqlQuery), err)
	}
	return nil
}

// ListAvailableRequestProperties возвращает объекты, доступные пользователю для создания заявки.
// Правила: владелец может по своим объявлениям, арендатор — только по accepted-договорам.
func (db *DB) ListAvailableRequestProperties(ctx context.Context, userID int) ([]AvailableRequestPropertyRow, error) {
	ownerRows, err := db.Pool.Query(ctx, `SELECT id FROM properties WHERE user_id = $1 ORDER BY id`, userID)
	if err != nil {
		return nil, err
	}
	ownerPropertyIDs := make([]int, 0)
	for ownerRows.Next() {
		var id int
		if err := ownerRows.Scan(&id); err != nil {
			ownerRows.Close()
			return nil, err
		}
		ownerPropertyIDs = append(ownerPropertyIDs, id)
	}
	ownerRows.Close()
	if err := ownerRows.Err(); err != nil {
		return nil, err
	}

	contractRows, err := db.Pool.Query(ctx, `
		SELECT id, property_id
		FROM contracts
		WHERE tenant_id = $1 AND status = 'accepted'
		ORDER BY id
	`, userID)
	if err != nil {
		return nil, err
	}
	tenantContractIDs := make([]int, 0)
	tenantPropertyIDs := make([]int, 0)
	for contractRows.Next() {
		var cid, pid int
		if err := contractRows.Scan(&cid, &pid); err != nil {
			contractRows.Close()
			return nil, err
		}
		tenantContractIDs = append(tenantContractIDs, cid)
		tenantPropertyIDs = append(tenantPropertyIDs, pid)
	}
	contractRows.Close()
	if err := contractRows.Err(); err != nil {
		return nil, err
	}

	roleType := "none"
	switch {
	case len(ownerPropertyIDs) > 0 && len(tenantContractIDs) > 0:
		roleType = "owner_and_tenant"
	case len(ownerPropertyIDs) > 0:
		roleType = "owner"
	case len(tenantContractIDs) > 0:
		roleType = "tenant"
	}
	log.Printf("[available-properties] currentUserId=%d role=%s contracts=%v tenantPropertyIDs=%v ownerPropertyIDs=%v",
		userID, roleType, tenantContractIDs, tenantPropertyIDs, ownerPropertyIDs)

	// Объединяем owner + tenant accepted, чтобы объект не терялся, если у пользователя сразу две роли.
	rows, err := db.Pool.Query(ctx, `
		WITH allowed AS (
			SELECT p.id FROM properties p WHERE p.user_id = $1
			UNION
			SELECT DISTINCT c.property_id
			FROM contracts c
			WHERE c.tenant_id = $1 AND c.status = 'accepted'
		)
		SELECT
			p.id,
			p.title,
			(SELECT NULLIF(TRIM(pi.image_url), '') FROM property_images pi WHERE pi.property_id = p.id ORDER BY pi.id ASC LIMIT 1) AS photo,
			p.address,
			p.city,
			p.district
		FROM properties p
		INNER JOIN allowed a ON a.id = p.id
		ORDER BY p.updated_at DESC, p.id DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]AvailableRequestPropertyRow, 0)
	for rows.Next() {
		var r AvailableRequestPropertyRow
		var photo sql.NullString
		if err := rows.Scan(&r.ID, &r.Title, &photo, &r.Address, &r.City, &r.District); err != nil {
			return nil, err
		}
		if photo.Valid {
			s := strings.TrimSpace(photo.String)
			if s != "" {
				r.Photo = &s
			}
		}
		out = append(out, r)
	}
	log.Printf("[available-properties] currentUserId=%d finalPropertyIDs=%v total=%d", userID, extractAvailablePropertyIDs(out), len(out))
	return out, rows.Err()
}

// GetRequestPropertyAccess проверяет существование объекта и право пользователя создавать по нему заявку.
func (db *DB) GetRequestPropertyAccess(ctx context.Context, userID, propertyID int) (exists bool, allowed bool, err error) {
	var anyID int
	err = db.Pool.QueryRow(ctx, `SELECT id FROM properties WHERE id = $1`, propertyID).Scan(&anyID)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, false, nil
	}
	if err != nil {
		return false, false, err
	}

	var can int
	err = db.Pool.QueryRow(ctx, `
		SELECT 1
		WHERE EXISTS (
			SELECT 1
			FROM properties p
			WHERE p.id = $1 AND p.user_id = $2
		) OR EXISTS (
			SELECT 1
			FROM contracts c
			WHERE c.property_id = $1 AND c.tenant_id = $2 AND c.status = 'accepted'
		)
	`, propertyID, userID).Scan(&can)
	if errors.Is(err, pgx.ErrNoRows) {
		return true, false, nil
	}
	if err != nil {
		return true, false, err
	}
	return true, true, nil
}

func extractAvailablePropertyIDs(items []AvailableRequestPropertyRow) []int {
	out := make([]int, 0, len(items))
	for _, it := range items {
		out = append(out, it.ID)
	}
	return out
}

func decodeStringJSONArray(raw []byte) []string {
	if len(raw) == 0 {
		return []string{}
	}
	var out []string
	if err := json.Unmarshal(raw, &out); err != nil {
		return []string{}
	}
	if out == nil {
		return []string{}
	}
	return out
}

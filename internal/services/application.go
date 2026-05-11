package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"rentora/backend/internal/models"
	"rentora/backend/internal/repository"
	aiSvc "rentora/backend/internal/services/ai"
)

// Сервис для заявок.
type ApplicationService struct {
	repo     *repository.DB
	analyzer aiSvc.PriorityAnalyzer
}

// Ошибка валидации тела заявки.
var ErrInvalidApplicationInput = errors.New("invalid application input")
var ErrRequestPropertyForbidden = errors.New("request property forbidden")
var ErrRequestDecisionForbidden = errors.New("request decision forbidden")
var ErrRequestDecisionNotFound = errors.New("request decision not found")
var ErrRequestDecisionInvalidStatus = errors.New("request decision invalid status")
var ErrRequestDecisionInvalidResolution = errors.New("request decision invalid resolution")
var ErrRequestExpenseForbidden = errors.New("request expense forbidden")
var ErrRequestExpenseInvalidScenario = errors.New("request expense invalid scenario")
var ErrRequestExpenseInvalidAmount = errors.New("request expense invalid amount")
var ErrConfirmTenantExpensesForbidden = errors.New("confirm tenant expenses forbidden")
var ErrConfirmTenantExpensesNotFound = errors.New("confirm tenant expenses not found")
var ErrConfirmTenantExpensesInvalidResolution = errors.New("confirm tenant expenses invalid resolution")
var ErrConfirmTenantExpensesNoExpenses = errors.New("confirm tenant expenses no expenses")
var ErrConfirmTenantExpensesWrongStatus = errors.New("confirm tenant expenses wrong status")
var ErrConfirmTenantExpensesAlready = errors.New("confirm tenant expenses already")
var ErrCompleteOwnerNotFound = errors.New("complete owner resolution not found")
var ErrCompleteOwnerForbidden = errors.New("complete owner resolution forbidden")
var ErrCompleteOwnerInvalidResolution = errors.New("complete owner resolution invalid resolution")
var ErrCompleteOwnerWrongStatus = errors.New("complete owner resolution wrong status")
var ErrCompleteOwnerAlreadyDone = errors.New("complete owner resolution already done")

// Конструктор сервиса заявок.
func NewApplicationService(repo *repository.DB, analyzer aiSvc.PriorityAnalyzer) *ApplicationService {
	if analyzer == nil {
		analyzer = aiSvc.NewMockPriorityAnalyzer()
	}
	return &ApplicationService{
		repo:     repo,
		analyzer: analyzer,
	}
}

// ListProfileRequests — без изменения запросов к БД: ListApplicationsByUser + ListApplicationsForOwnerProperties.
// Дальше только разделение в памяти на два слайса: activeRequests (status !== completed), archivedRequests (status === completed).
func (s *ApplicationService) ListProfileRequests(ctx context.Context, userID int, bucket string) (*models.ProfileRequestsResponse, error) {
	_ = bucket

	myRows, err := s.repo.ListApplicationsByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	ownerPropertyIDs, ownerRows, err := s.repo.ListApplicationsForOwnerProperties(ctx, userID)
	if err != nil {
		return nil, err
	}
	sort.Slice(myRows, func(i, j int) bool {
		return myRows[i].CreatedAt.After(myRows[j].CreatedAt)
	})
	sort.Slice(ownerRows, func(i, j int) bool {
		return ownerRows[i].CreatedAt.After(ownerRows[j].CreatedAt)
	})
	log.Printf("[requests-owner] currentUserId=%d ownerPropertyIDs=%v", userID, ownerPropertyIDs)
	log.Printf("[requests-owner] currentUserId=%d ownerRequestIDs=%v", userID, extractOwnerRequestIDs(ownerRows))

	log.Printf("[debug][profile-requests] GET currentUserId=%d: ListApplicationsByUser фильтр SQL — WHERE applications.user_id = applicant (тот же userId); ListApplicationsForOwnerProperties — WHERE properties.user_id = владелец объекта", userID)
	log.Printf("[debug][profile-requests] сырые строки из БД — myRows n=%d: %s", len(myRows), formatApplicationRowsDebug(myRows))
	log.Printf("[debug][profile-requests] сырые строки из БД — ownerRows n=%d: %s", len(ownerRows), formatPropertyRequestRowsDebug(ownerRows))

	var myActive, myArchived []models.ProfileRequestItem
	for _, r := range myRows {
		it := mapApplicationRowToProfileRequestItem(r)
		it.IsArchived = profileStatusIsCompleted(it.Status)
		if profileStatusIsCompleted(it.Status) {
			myArchived = append(myArchived, it)
		} else {
			myActive = append(myActive, it)
		}
	}

	var propActive, propArchived []models.PropertyRequestItem
	for _, r := range ownerRows {
		it := mapPropertyRequestRowToItem(r)
		it.IsArchived = profileStatusIsCompleted(it.Status)
		if profileStatusIsCompleted(it.Status) {
			propArchived = append(propArchived, it)
		} else {
			propActive = append(propActive, it)
		}
	}

	activeMerged := mergeProfileRequestsForJSON(myActive, propActive)
	archivedMerged := mergeProfileRequestsForJSON(myArchived, propArchived)

	log.Printf("[debug][profile-requests] после split по status: activeRequests id (my затем props)=[%s] | archivedRequests id=[%s]",
		joinSplitProfileIDs(myActive, propActive), joinSplitProfileIDs(myArchived, propArchived))
	log.Printf("[profile-requests] userId=%d active_total=%d archived_total=%d (my: active %d arch %d, props: active %d arch %d)",
		userID, len(activeMerged), len(archivedMerged), len(myActive), len(myArchived), len(propActive), len(propArchived))

	return &models.ProfileRequestsResponse{
		ActiveRequests:   activeMerged,
		ArchivedRequests: archivedMerged,
	}, nil
}

func mergeProfileRequestsForJSON(my []models.ProfileRequestItem, props []models.PropertyRequestItem) []models.ProfileRequestsEntry {
	out := make([]models.ProfileRequestsEntry, 0, len(my)+len(props))
	for i := range my {
		out = append(out, my[i])
	}
	for i := range props {
		out = append(out, props[i])
	}
	return out
}

func formatApplicationRowsDebug(rows []repository.ApplicationRow) string {
	if len(rows) == 0 {
		return "<empty>"
	}
	parts := make([]string, 0, len(rows))
	for _, r := range rows {
		parts = append(parts, fmt.Sprintf("%d:%s", r.ID, r.Status))
	}
	return strings.Join(parts, ", ")
}

func formatPropertyRequestRowsDebug(rows []repository.PropertyRequestRow) string {
	if len(rows) == 0 {
		return "<empty>"
	}
	parts := make([]string, 0, len(rows))
	for _, r := range rows {
		parts = append(parts, fmt.Sprintf("%d:%s", r.ID, r.Status))
	}
	return strings.Join(parts, ", ")
}

func joinSplitProfileIDs(my []models.ProfileRequestItem, props []models.PropertyRequestItem) string {
	parts := make([]string, 0, len(my)+len(props))
	for _, x := range my {
		parts = append(parts, fmt.Sprintf("%d:%s", x.ID, x.Status))
	}
	for _, x := range props {
		parts = append(parts, fmt.Sprintf("%d:%s", x.ID, x.Status))
	}
	return strings.Join(parts, ", ")
}

// normalizeRequestStatusForAPI: пустой/битый статус из БД → pending (иначе фронт и фильтр activeRequests ломаются).
func normalizeRequestStatusForAPI(status string) string {
	s := strings.TrimSpace(status)
	if s == "" {
		return models.RequestStatusPending
	}
	return s
}

// profileStatusIsCompleted: archivedRequests = status === "completed"; activeRequests = всё остальное после нормализации.
func profileStatusIsCompleted(status string) bool {
	return strings.EqualFold(strings.TrimSpace(status), models.RequestStatusCompleted)
}

func applicationIsArchived(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case models.RequestStatusCompleted:
		return true
	default:
		return false
	}
}

// rowIsArchived: флаг из БД; при старых данных без колонки — по финальному статусу.
func rowIsArchived(dbValue bool, status string) bool {
	if dbValue {
		return true
	}
	return applicationIsArchived(status)
}

func propertyCardFromApplicationRow(r repository.ApplicationRow) *models.Property {
	if r.PropertyID <= 0 {
		return nil
	}
	photos := r.PropertyPhotos
	if photos == nil {
		photos = []string{}
	}
	return &models.Property{
		ID:           r.PropertyID,
		Title:        r.PropertyTitle,
		Price:        r.PropertyPrice,
		PropertyType: r.PropertyTypeName,
		Rooms:        r.PropertyRooms,
		TotalArea:    r.PropertyTotalArea,
		City:         r.PropertyCity,
		District:     r.PropertyDistrict,
		Photos:       photos,
	}
}

func propertyCardFromPropertyRequestRow(r repository.PropertyRequestRow) *models.Property {
	if r.PropertyID <= 0 {
		return nil
	}
	photos := r.PropertyPhotos
	if photos == nil {
		photos = []string{}
	}
	return &models.Property{
		ID:           r.PropertyID,
		Title:        r.PropertyTitle,
		Price:        r.PropertyPrice,
		PropertyType: r.PropertyTypeName,
		Rooms:        r.PropertyRooms,
		TotalArea:    r.PropertyTotalArea,
		City:         r.PropertyCity,
		District:     r.PropertyDistrict,
		Photos:       photos,
	}
}

func mapApplicationRowToProfileRequestItem(r repository.ApplicationRow) models.ProfileRequestItem {
	st := normalizeRequestStatusForAPI(r.Status)
	arch := profileStatusIsCompleted(st)
	return models.ProfileRequestItem{
		ID:               r.ID,
		Title:            r.Title,
		Description:      r.Description,
		Category:         r.Category,
		Status:           st,
		Priority:         r.Priority,
		PriorityStatus:   r.PriorityStatus,
		PriorityScore:    r.PriorityScore,
		PriorityReason:   r.PriorityReason,
		ResolutionType:   r.ResolutionType,
		ResolutionTypeRaw:  r.ResolutionType,
		RequesterID:      r.UserID,
		RequesterName:    r.RequesterName,
		RequestPhotos:    r.RequestPhotos,
		ExpenseAmount:    r.ExpenseAmount,
		ExpenseComment:   r.ExpenseComment,
		ExpensePhotos:    r.ExpensePhotos,
		ExpensesSubmitted: r.ExpensesSubmitted,
		CreatedAt:        r.CreatedAt,
		PropertyID:       r.PropertyID,
		PropertyTitle:    r.PropertyTitle,
		PropertyPhoto:    r.PropertyPhoto,
		PropertyAddress:  r.PropertyAddress,
		PropertyCity:     r.PropertyCity,
		PropertyDistrict: r.PropertyDistrict,
		Property:         propertyCardFromApplicationRow(r),
		TenantExpensesConfirmedAt: nullTimeToPtr(r.TenantExpensesConfirmedAt),
		IsArchived:       arch,
	}
}

func mapPropertyRequestRowToItem(r repository.PropertyRequestRow) models.PropertyRequestItem {
	st := normalizeRequestStatusForAPI(r.Status)
	arch := profileStatusIsCompleted(st)
	it := models.PropertyRequestItem{
		ID:               r.ID,
		Title:            r.Title,
		Description:      r.Description,
		Status:           st,
		RequesterID:      r.RequesterID,
		RequesterName:    r.RequesterName,
		PropertyOwnerID:  r.PropertyOwnerID,
		TenantExpensesConfirmedAt: nullTimeToPtr(r.TenantExpensesConfirmedAt),
		IsArchived:       arch,
	}
	if property := propertyCardFromPropertyRequestRow(r); property != nil {
		it.Property = property
	}
	return it
}

func extractOwnerRequestIDs(rows []repository.PropertyRequestRow) []int {
	out := make([]int, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.ID)
	}
	return out
}

// ListAvailableProperties возвращает объекты, доступные пользователю для создания заявки.
func (s *ApplicationService) ListAvailableProperties(ctx context.Context, userID int) ([]models.AvailableRequestPropertyItem, error) {
	rows, err := s.repo.ListAvailableRequestProperties(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]models.AvailableRequestPropertyItem, 0, len(rows))
	for _, r := range rows {
		out = append(out, models.AvailableRequestPropertyItem{
			ID:       r.ID,
			Title:    r.Title,
			Photo:    r.Photo,
			Address:  r.Address,
			City:     r.City,
			District: r.District,
		})
	}
	return out, nil
}

// CreateRequest создает новую заявку со статусом pending.
func (s *ApplicationService) CreateRequest(ctx context.Context, userID int, in models.CreateRequestBody, requestPhotos []string) (*models.CreateRequestResponse, error) {
	if in.PropertyID == nil || *in.PropertyID < 1 {
		return nil, ErrInvalidApplicationInput
	}
	title := strings.TrimSpace(in.Title)
	desc := strings.TrimSpace(in.Description)
	category := strings.TrimSpace(in.Category)
	if title == "" || desc == "" || category == "" {
		return nil, ErrInvalidApplicationInput
	}
	exists, allowed, err := s.repo.GetRequestPropertyAccess(ctx, userID, *in.PropertyID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, repository.ErrPropertyNotFound
	}
	if !allowed {
		return nil, ErrRequestPropertyForbidden
	}

	row, err := s.repo.CreateApplication(ctx, userID, *in.PropertyID, title, desc, category, requestPhotos)
	if err != nil {
		return nil, err
	}
	statusOut := normalizeRequestStatusForAPI(row.Status)
	go s.launchPriorityAnalysis(row.ID, title, desc, category)
	log.Printf("[AI priority] requestId=%d queued", row.ID)
	log.Printf("[requests-create] requestId=%d status=%q (ожидается %q для попадания в activeRequests)", row.ID, statusOut, models.RequestStatusPending)
	log.Printf("[debug][requests-create] в БД: application id=%d applicant_user_id=%d (= JWT при создании) property_id=%d status=%q | в GET /profile/requests эта заявка попадёт в ListApplicationsByUser только если JWT user_id совпадает с applicant_user_id (или в owner-список, если вы владелец объекта)",
		row.ID, row.UserID, row.PropertyID, statusOut)

	return &models.CreateRequestResponse{
		ID:             row.ID,
		PropertyID:     row.PropertyID,
		Title:          row.Title,
		Description:    row.Description,
		Category:       row.Category,
		Status:         statusOut,
		Priority:       row.Priority,
		PriorityStatus: row.PriorityStatus,
		PriorityScore:  row.PriorityScore,
		PriorityReason: row.PriorityReason,
		ResolutionType: row.ResolutionType,
		RequestPhotos:  row.RequestPhotos,
		ExpenseAmount:  row.ExpenseAmount,
		ExpenseComment: row.ExpenseComment,
		ExpensePhotos:  row.ExpensePhotos,
		ExpensesSubmitted: row.ExpensesSubmitted,
		CreatedAt:      row.CreatedAt,
		PropertyOwnerID: row.PropertyOwnerID,
	}, nil
}

func (s *ApplicationService) DecideRequest(ctx context.Context, userID, requestID int, resolutionType string) (*models.RequestDecisionResponse, error) {
	resolutionType = strings.ToLower(strings.TrimSpace(resolutionType))
	log.Printf("[requests-decision] requestId=%d currentUserId=%d resolutionType=%s", requestID, userID, resolutionType)
	var nextStatus string
	switch resolutionType {
	case models.RequestResolutionTypeOwner:
		nextStatus = repository.ApplicationStatusOwnerResolves
	case models.RequestResolutionTypeTenant:
		nextStatus = repository.ApplicationStatusTenantResolves
	default:
		return nil, ErrRequestDecisionInvalidResolution
	}

	info, err := s.repo.GetRequestDecisionInfo(ctx, requestID)
	if err != nil {
		if errors.Is(err, repository.ErrRequestNotFound) {
			return nil, ErrRequestDecisionNotFound
		}
		return nil, err
	}
	if info.PropertyOwnerID != userID {
		log.Printf("[requests-decision] requestId=%d currentUserId=%d forbidden propertyOwnerId=%d", requestID, userID, info.PropertyOwnerID)
		return nil, ErrRequestDecisionForbidden
	}
	if info.Status != repository.ApplicationStatusPending {
		log.Printf("[requests-decision] requestId=%d currentUserId=%d invalid_status=%s", requestID, userID, info.Status)
		return nil, ErrRequestDecisionInvalidStatus
	}
	if err := s.repo.ApplyRequestDecision(ctx, requestID, nextStatus, resolutionType); err != nil {
		return nil, err
	}
	log.Printf("[requests-decision] requestId=%d currentUserId=%d updated status=%s resolutionType=%s", requestID, userID, nextStatus, resolutionType)
	return &models.RequestDecisionResponse{
		ID:               info.ID,
		Title:            info.Title,
		Description:      info.Description,
		Status:           nextStatus,
		ResolutionType:   resolutionType,
		Priority:         info.Priority,
		PriorityReason:   info.PriorityReason,
		CreatedAt:        info.CreatedAt,
		PropertyID:       info.PropertyID,
		PropertyTitle:    info.PropertyTitle,
		PropertyPhoto:    info.PropertyPhoto,
		PropertyAddress:  info.PropertyAddress,
		PropertyCity:     info.PropertyCity,
		PropertyDistrict: info.PropertyDistrict,
		RequesterID:      info.RequesterID,
		RequesterName:    info.RequesterName,
		PropertyOwnerID:  info.PropertyOwnerID,
		ExpenseAmount:    info.ExpenseAmount,
		ExpenseComment:   info.ExpenseComment,
		ExpensePhotos:    info.ExpensePhotos,
		ExpensesSubmitted: info.ExpensesSubmitted,
	}, nil
}

func (s *ApplicationService) SubmitRequestExpense(ctx context.Context, userID, requestID int, expenseAmount float64, expenseComment string, expensePhotos []string) (*models.RequestExpenseResponse, error) {
	if expenseAmount < 0 {
		return nil, ErrRequestExpenseInvalidAmount
	}
	info, err := s.repo.GetRequestDecisionInfo(ctx, requestID)
	if err != nil {
		if errors.Is(err, repository.ErrRequestNotFound) {
			return nil, ErrRequestDecisionNotFound
		}
		return nil, err
	}
	if info.RequesterID != userID {
		return nil, ErrRequestExpenseForbidden
	}
	if !(info.Status == repository.ApplicationStatusTenantResolves || (info.ResolutionType != nil && strings.EqualFold(*info.ResolutionType, models.RequestResolutionTypeTenant))) {
		return nil, ErrRequestExpenseInvalidScenario
	}

	comment := strings.TrimSpace(expenseComment)
	nextStatus := repository.ApplicationStatusTenantResolves
	if err := s.repo.ApplyRequestExpense(ctx, requestID, expenseAmount, comment, expensePhotos, nextStatus); err != nil {
		return nil, err
	}

	updated, err := s.repo.GetRequestDecisionInfo(ctx, requestID)
	if err != nil {
		return nil, err
	}
	return mapRequestExpenseResponse(updated), nil
}

func (s *ApplicationService) ConfirmTenantExpenses(ctx context.Context, ownerUserID, requestID int) (*models.ConfirmTenantExpensesResponse, error) {
	info, err := s.repo.GetRequestDecisionInfo(ctx, requestID)
	if err != nil {
		if errors.Is(err, repository.ErrRequestNotFound) {
			return nil, ErrConfirmTenantExpensesNotFound
		}
		return nil, err
	}
	if info.PropertyOwnerID != ownerUserID {
		return nil, ErrConfirmTenantExpensesForbidden
	}
	if info.ResolutionType == nil || !strings.EqualFold(strings.TrimSpace(*info.ResolutionType), models.RequestResolutionTypeTenant) {
		return nil, ErrConfirmTenantExpensesInvalidResolution
	}
	if !info.ExpensesSubmitted {
		return nil, ErrConfirmTenantExpensesNoExpenses
	}
	if info.Status == repository.ApplicationStatusCompleted {
		return nil, ErrConfirmTenantExpensesAlready
	}
	if info.TenantExpensesConfirmedAt.Valid {
		return nil, ErrConfirmTenantExpensesAlready
	}
	if info.Status != repository.ApplicationStatusTenantResolves {
		return nil, ErrConfirmTenantExpensesWrongStatus
	}

	saved, err := s.repo.ConfirmTenantExpenses(ctx, requestID)
	if err != nil {
		return nil, err
	}
	if saved == nil {
		return nil, ErrConfirmTenantExpensesWrongStatus
	}
	if !saved.IsArchived || !strings.EqualFold(strings.TrimSpace(saved.Status), models.RequestStatusCompleted) {
		log.Printf("[requests-confirm-expenses] invariant requestId=%d RETURNING status=%q is_archived=%v", requestID, saved.Status, saved.IsArchived)
	}

	after, err := s.repo.GetRequestDecisionInfo(ctx, requestID)
	if err != nil {
		return nil, err
	}
	if after.TenantExpensesConfirmedAt.Valid {
		log.Printf("[requests-confirm-expenses] requestId=%d ownerUserId=%d status=%s confirmedAt=%v", requestID, ownerUserID, after.Status, after.TenantExpensesConfirmedAt.Time)
	} else {
		log.Printf("[requests-confirm-expenses] requestId=%d ownerUserId=%d status=%s confirmedAt=<none>", requestID, ownerUserID, after.Status)
	}

	resp := mapConfirmTenantExpensesResponse(after)
	st := strings.TrimSpace(saved.Status)
	if strings.EqualFold(st, models.RequestStatusCompleted) {
		st = models.RequestStatusCompleted
	}
	reqInfo := models.ConfirmTenantExpenseRequestInfo{
		ID:         saved.ID,
		Status:     st,
		IsArchived: saved.IsArchived,
	}
	resp.Request = reqInfo
	resp.ID = reqInfo.ID
	resp.Status = reqInfo.Status
	resp.IsArchived = reqInfo.IsArchived
	if !after.IsArchived {
		log.Printf("[requests-confirm-expenses] warn requestId=%d reload is_archived false after confirm", requestID)
	}
	if !strings.EqualFold(strings.TrimSpace(after.Status), models.RequestStatusCompleted) {
		log.Printf("[requests-confirm-expenses] warn requestId=%d reload status %q want completed", requestID, after.Status)
	}
	return resp, nil
}

// CompleteOwnerResolution завершает сценарий «устраняет владелец»: owner_resolves → completed.
func (s *ApplicationService) CompleteOwnerResolution(ctx context.Context, ownerUserID, requestID int) (*models.PropertyRequestItem, error) {
	return s.CompleteOwnerRequest(ctx, ownerUserID, requestID)
}

// CompleteOwnerRequest завершает owner flow: owner_resolves → completed в БД.
func (s *ApplicationService) CompleteOwnerRequest(ctx context.Context, ownerUserID, requestID int) (*models.PropertyRequestItem, error) {
	info, err := s.repo.GetRequestDecisionInfo(ctx, requestID)
	if err != nil {
		if errors.Is(err, repository.ErrRequestNotFound) {
			return nil, ErrCompleteOwnerNotFound
		}
		return nil, err
	}
	if info.PropertyOwnerID != ownerUserID {
		return nil, ErrCompleteOwnerForbidden
	}
	if info.ResolutionType == nil || !strings.EqualFold(strings.TrimSpace(*info.ResolutionType), models.RequestResolutionTypeOwner) {
		return nil, ErrCompleteOwnerInvalidResolution
	}
	if info.Status == repository.ApplicationStatusCompleted {
		return nil, ErrCompleteOwnerAlreadyDone
	}
	if info.Status != repository.ApplicationStatusOwnerResolves {
		return nil, ErrCompleteOwnerWrongStatus
	}

	saved, err := s.repo.CompleteOwnerRequestToCompleted(ctx, requestID, ownerUserID)
	if err != nil {
		return nil, err
	}
	if saved == nil {
		return nil, ErrCompleteOwnerWrongStatus
	}
	if !strings.EqualFold(strings.TrimSpace(saved.Status), models.RequestStatusCompleted) {
		log.Printf("[requests-complete-owner-request] warn requestId=%d RETURNING status=%q", requestID, saved.Status)
	}

	row, err := s.repo.GetPropertyRequestForOwner(ctx, ownerUserID, requestID)
	if err != nil {
		if errors.Is(err, repository.ErrRequestNotFound) {
			return nil, ErrCompleteOwnerNotFound
		}
		return nil, err
	}
	it := mapPropertyRequestRowToItem(*row)
	log.Printf("[requests-complete-owner-request] requestId=%d ownerUserId=%d status=%s isArchived=%v", requestID, ownerUserID, it.Status, it.IsArchived)
	return &it, nil
}

func nullTimeToPtr(nt sql.NullTime) *time.Time {
	if !nt.Valid {
		return nil
	}
	t := nt.Time
	return &t
}

func mapRequestExpenseResponse(info *repository.RequestDecisionRow) *models.RequestExpenseResponse {
	resType := ""
	if info.ResolutionType != nil {
		resType = strings.TrimSpace(*info.ResolutionType)
	}
	var alt *string
	if resType != "" {
		v := resType
		alt = &v
	}
	amount := 0.0
	if info.ExpenseAmount != nil {
		amount = *info.ExpenseAmount
	}
	comment := ""
	if info.ExpenseComment != nil {
		comment = strings.TrimSpace(*info.ExpenseComment)
	}
	photos := info.ExpensePhotos
	if photos == nil {
		photos = []string{}
	}
	return &models.RequestExpenseResponse{
		ID:                 info.ID,
		Title:              info.Title,
		Description:        info.Description,
		Status:             info.Status,
		ResolutionType:     resType,
		ResolutionTypeAlt:  alt,
		Priority:           info.Priority,
		PriorityReason:     info.PriorityReason,
		ExpenseAmount:      amount,
		ExpenseComment:     comment,
		ExpensePhotos:      photos,
		ExpensesSubmitted:  info.ExpensesSubmitted,
		CreatedAt:          info.CreatedAt,
		PropertyID:         info.PropertyID,
		PropertyTitle:      info.PropertyTitle,
		PropertyPhoto:      info.PropertyPhoto,
		PropertyAddress:    info.PropertyAddress,
		PropertyCity:       info.PropertyCity,
		PropertyDistrict:   info.PropertyDistrict,
		RequesterID:        info.RequesterID,
		RequesterName:      info.RequesterName,
		PropertyOwnerID:    info.PropertyOwnerID,
		IsArchived:         rowIsArchived(info.IsArchived, info.Status),
	}
}

func mapConfirmTenantExpensesResponse(info *repository.RequestDecisionRow) *models.ConfirmTenantExpensesResponse {
	resType := ""
	if info.ResolutionType != nil {
		resType = strings.TrimSpace(*info.ResolutionType)
	}
	var alt *string
	if resType != "" {
		v := resType
		alt = &v
	}
	amount := 0.0
	if info.ExpenseAmount != nil {
		amount = *info.ExpenseAmount
	}
	comment := ""
	if info.ExpenseComment != nil {
		comment = strings.TrimSpace(*info.ExpenseComment)
	}
	photos := info.ExpensePhotos
	if photos == nil {
		photos = []string{}
	}
	var confirmedAt time.Time
	if info.TenantExpensesConfirmedAt.Valid {
		confirmedAt = info.TenantExpensesConfirmedAt.Time
	}
	return &models.ConfirmTenantExpensesResponse{
		ID:                 info.ID,
		Title:              info.Title,
		Description:        info.Description,
		Status:             info.Status,
		ResolutionType:   resType,
		ResolutionTypeAlt:  alt,
		Priority:           info.Priority,
		PriorityReason:     info.PriorityReason,
		ExpenseAmount:      amount,
		ExpenseComment:     comment,
		ExpensePhotos:      photos,
		ExpensesSubmitted:  info.ExpensesSubmitted,
		ConfirmedAt:        confirmedAt,
		CreatedAt:          info.CreatedAt,
		PropertyID:         info.PropertyID,
		PropertyTitle:      info.PropertyTitle,
		PropertyPhoto:      info.PropertyPhoto,
		PropertyAddress:    info.PropertyAddress,
		PropertyCity:       info.PropertyCity,
		PropertyDistrict:   info.PropertyDistrict,
		RequesterID:        info.RequesterID,
		RequesterName:      info.RequesterName,
		PropertyOwnerID:    info.PropertyOwnerID,
		IsArchived:         rowIsArchived(info.IsArchived, info.Status),
	}
}

func (s *ApplicationService) launchPriorityAnalysis(requestID int, title, description, category string) {
	log.Printf("[AI priority] requestId=%d start", requestID)
	log.Printf("[AI priority] requestId=%d input title=%q description=%q category=%q", requestID, title, description, category)

	log.Printf("[AI priority] requestId=%d analyze_call begin", requestID)
	analyzeCtx, analyzeCancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer analyzeCancel()
	res, rawResponse, err := s.analyzer.Analyze(analyzeCtx, title, description, category)
	priorityStatus := models.RequestPriorityStatusReady
	if err != nil {
		log.Printf("[AI priority] requestId=%d ai_error=%v", requestID, err)
		if strings.TrimSpace(rawResponse) != "" {
			log.Printf("[AI priority] requestId=%d ai_raw_response_on_error=%q", requestID, rawResponse)
		}
		res = aiSvc.DefaultPriorityResult()
		priorityStatus = models.RequestPriorityStatusFallback
	} else {
		log.Printf("[AI priority] requestId=%d ai_result priority=%s score=%.2f reason=%q", requestID, res.Priority, res.PriorityScore, res.PriorityReason)
		log.Printf("[AI priority] requestId=%d ai_raw_response=%q", requestID, rawResponse)
		log.Printf("[AI priority] requestId=%d ai_parsed priority=%s score=%.2f reason=%q", requestID, res.Priority, res.PriorityScore, res.PriorityReason)
	}

	updateCtx, updateCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer updateCancel()
	if err := s.repo.UpdateApplicationPriority(updateCtx, requestID, res.Priority, priorityStatus, res.PriorityScore, res.PriorityReason); err != nil {
		log.Printf("[AI priority] requestId=%d update_error=%v priority=%s priorityStatus=%s priorityScore=%.2f priorityReason=%q",
			requestID, err, res.Priority, priorityStatus, res.PriorityScore, res.PriorityReason)
		return
	}
	log.Printf("[AI priority] requestId=%d update_ok priority=%s status=%s score=%.2f reason=%q",
		requestID, res.Priority, priorityStatus, res.PriorityScore, res.PriorityReason)
}

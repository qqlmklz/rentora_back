package models

import "time"

const (
	RequestPriorityLow    = "low"
	RequestPriorityMedium = "medium"
	RequestPriorityHigh   = "high"

	RequestPriorityStatusPending  = "pending"
	RequestPriorityStatusReady    = "ready"
	RequestPriorityStatusFallback = "fallback"

	RequestResolutionTypeOwner  = "owner"
	RequestResolutionTypeTenant = "tenant"

	// Стартовый статус новой заявки (activeRequests: всё, что не completed).
	RequestStatusPending = "pending"

	// Финальные статусы заявки (попадают в архив при выдаче профиля).
	RequestStatusCompleted = "completed"
	RequestStatusResolved  = "resolved"
)

// Тело запроса для POST /api/requests.
type CreateRequestBody struct {
	PropertyID  *int   `json:"propertyId" binding:"required"`
	Title       string `json:"title" binding:"required"`
	Description string `json:"description" binding:"required"`
	Category    string `json:"category" binding:"required"`
}

// ProfileRequestsEntry — допустимый элемент массивов activeRequests и archivedRequests в GET /api/profile/requests.
// В JSON это гетерогенный массив объектов без поля-типа: каждый элемент либо ProfileRequestItem, либо PropertyRequestItem
// (различие по набору полей: у «моих» заявок есть category и priorityStatus/priorityScore; у заявок по моим объектам — propertyOwnerId).
type ProfileRequestsEntry interface {
	profileRequestsEntry()
}

func (ProfileRequestItem) profileRequestsEntry() {}
func (PropertyRequestItem) profileRequestsEntry() {}

// ProfileRequestItem — заявка, где текущий пользователь является заявителем (арендатор по объекту).
type ProfileRequestItem struct {
	ID               int       `json:"id"`
	Title            string    `json:"title"`
	Description      string    `json:"description"`
	Category         string    `json:"category"`
	Status           string    `json:"status"`
	Priority         string    `json:"priority"`
	PriorityStatus   string    `json:"priorityStatus"`
	PriorityScore    float64   `json:"priorityScore"`
	PriorityReason   string    `json:"priorityReason"`
	ResolutionType   *string   `json:"resolutionType,omitempty"`
	ResolutionTypeRaw *string  `json:"resolution_type,omitempty"`
	RequesterID      int       `json:"requesterId"`
	RequesterName    string    `json:"requesterName"`
	RequestPhotos    []string  `json:"request_photos"`
	ExpenseAmount    *float64  `json:"expenseAmount,omitempty"`
	ExpenseComment   *string   `json:"expenseComment,omitempty"`
	ExpensePhotos    []string  `json:"expensePhotos"`
	CreatedAt        time.Time `json:"createdAt"`
	PropertyID       int       `json:"propertyId"`
	PropertyTitle    string    `json:"propertyTitle"`
	PropertyPhoto    *string   `json:"propertyPhoto"`
	PropertyAddress  string    `json:"propertyAddress"`
	PropertyCity     string    `json:"propertyCity"`
	PropertyDistrict string    `json:"propertyDistrict"`
	Property         Property  `json:"property"`
	TenantExpensesConfirmedAt *time.Time `json:"confirmedAt,omitempty"`
	IsArchived       bool      `json:"isArchived"`
}

// PropertyRequestItem — заявка по объявлению, где текущий пользователь владелец объекта (не заявитель).
type PropertyRequestItem struct {
	ID               int       `json:"id"`
	Title            string    `json:"title"`
	Description      string    `json:"description"`
	Status           string    `json:"status"`
	Priority         string    `json:"priority"`
	PriorityReason   string    `json:"priorityReason"`
	ResolutionType   *string   `json:"resolutionType,omitempty"`
	ResolutionTypeRaw *string  `json:"resolution_type,omitempty"`
	RequestPhotos    []string  `json:"request_photos"`
	ExpenseAmount    *float64  `json:"expenseAmount,omitempty"`
	ExpenseComment   *string   `json:"expenseComment,omitempty"`
	ExpensePhotos    []string  `json:"expensePhotos"`
	CreatedAt        time.Time `json:"createdAt"`
	PropertyID       int       `json:"propertyId"`
	PropertyTitle    string    `json:"propertyTitle"`
	PropertyPhoto    *string   `json:"propertyPhoto"`
	PropertyAddress  string    `json:"propertyAddress"`
	PropertyCity     string    `json:"propertyCity"`
	PropertyDistrict string    `json:"propertyDistrict"`
	Property         Property  `json:"property"`
	RequesterID      int       `json:"requesterId"`
	RequesterName    string    `json:"requesterName"`
	PropertyOwnerID  int       `json:"propertyOwnerId"`
	TenantExpensesConfirmedAt *time.Time `json:"confirmedAt,omitempty"`
	IsArchived       bool      `json:"isArchived"`
}

// Секция списка заявок (активные или архивные).
type ProfileRequestsSection struct {
	MyRequests              []ProfileRequestItem  `json:"myRequests"`
	RequestsForMyProperties []PropertyRequestItem `json:"requestsForMyProperties"`
}

// ProfileRequestsResponse — тело 200 OK для GET /api/profile/requests (query bucket игнорируется).
//
// Верхний уровень JSON:
//
//	{ "activeRequests": [...], "archivedRequests": [...] }
//
// Оба ключа — массивы; пустые списки сериализуются как [].
// Элементы: ProfileRequestsEntry — сначала все ProfileRequestItem (мои заявки), затем все PropertyRequestItem (по моим объектам), внутри каждого списка порядок как в сервисе (обычно по дате создания заявки).
// У вложенного объекта недвижимости ключ JSON — "property", тип полей см. Property.
type ProfileRequestsResponse struct {
	ActiveRequests   []ProfileRequestsEntry `json:"activeRequests"`
	ArchivedRequests []ProfileRequestsEntry `json:"archivedRequests"`
}

// Ответ для POST /api/requests.
type CreateRequestResponse struct {
	ID             int       `json:"id"`
	PropertyID     int       `json:"propertyId"`
	Title          string    `json:"title"`
	Description    string    `json:"description"`
	Category       string    `json:"category"`
	Status         string    `json:"status"`
	Priority       string    `json:"priority"`
	PriorityStatus string    `json:"priorityStatus"`
	PriorityScore  float64   `json:"priorityScore"`
	PriorityReason string    `json:"priorityReason"`
	ResolutionType *string   `json:"resolutionType,omitempty"`
	RequestPhotos  []string  `json:"request_photos"`
	ExpenseAmount  *float64  `json:"expenseAmount,omitempty"`
	ExpenseComment *string   `json:"expenseComment,omitempty"`
	ExpensePhotos  []string  `json:"expensePhotos"`
	CreatedAt      time.Time `json:"createdAt"`
}

// Тело запроса для PATCH /api/requests/:id/decision.
type RequestDecisionBody struct {
	ResolutionType string `json:"resolutionType"`
}

// Ответ PATCH /api/requests/:id/decision.
type RequestDecisionResponse struct {
	ID               int       `json:"id"`
	Title            string    `json:"title"`
	Description      string    `json:"description"`
	Status           string    `json:"status"`
	ResolutionType   string    `json:"resolution_type"`
	Priority         string    `json:"priority"`
	PriorityReason   string    `json:"priority_reason"`
	CreatedAt        time.Time `json:"createdAt"`
	PropertyID       int       `json:"propertyId"`
	PropertyTitle    string    `json:"propertyTitle"`
	PropertyPhoto    *string   `json:"propertyPhoto"`
	PropertyAddress  string    `json:"propertyAddress"`
	PropertyCity     string    `json:"propertyCity"`
	PropertyDistrict string    `json:"propertyDistrict"`
	RequesterID      int       `json:"requesterId"`
	RequesterName    string    `json:"requesterName"`
	PropertyOwnerID  int       `json:"propertyOwnerId"`
	ExpenseAmount    *float64  `json:"expenseAmount,omitempty"`
	ExpenseComment   *string   `json:"expenseComment,omitempty"`
	ExpensePhotos    []string  `json:"expensePhotos"`
}

// Ответ PATCH /api/requests/:id/expense.
type RequestExpenseResponse struct {
	ID               int       `json:"id"`
	Title            string    `json:"title"`
	Description      string    `json:"description"`
	Status           string    `json:"status"`
	ResolutionType   string    `json:"resolution_type"`
	ResolutionTypeAlt *string  `json:"resolutionType,omitempty"`
	Priority         string    `json:"priority"`
	PriorityReason   string    `json:"priorityReason"`
	ExpenseAmount    float64   `json:"expenseAmount"`
	ExpenseComment   string    `json:"expenseComment"`
	ExpensePhotos    []string  `json:"expensePhotos"`
	CreatedAt        time.Time `json:"createdAt"`
	PropertyID       int       `json:"propertyId"`
	PropertyTitle    string    `json:"propertyTitle"`
	PropertyPhoto    *string   `json:"propertyPhoto"`
	PropertyAddress  string    `json:"propertyAddress"`
	PropertyCity     string    `json:"propertyCity"`
	PropertyDistrict string    `json:"propertyDistrict"`
	RequesterID      int       `json:"requesterId"`
	RequesterName    string    `json:"requesterName"`
	IsArchived       bool      `json:"isArchived"`
}

// Снимок заявки после confirm (значения из БД, для диагностики архива).
type ConfirmTenantExpenseRequestInfo struct {
	ID         int    `json:"id"`
	Status     string `json:"status"`
	IsArchived bool   `json:"isArchived"`
}

// Ответ POST /api/requests/:id/confirm-tenant-expenses.
type ConfirmTenantExpensesResponse struct {
	Request ConfirmTenantExpenseRequestInfo `json:"request"`
	ID               int        `json:"id"`
	Title            string     `json:"title"`
	Description      string     `json:"description"`
	Status           string     `json:"status"`
	ResolutionType   string     `json:"resolution_type"`
	ResolutionTypeAlt *string   `json:"resolutionType,omitempty"`
	Priority         string     `json:"priority"`
	PriorityReason   string     `json:"priorityReason"`
	ExpenseAmount    float64    `json:"expenseAmount"`
	ExpenseComment   string     `json:"expenseComment"`
	ExpensePhotos    []string   `json:"expensePhotos"`
	ConfirmedAt      time.Time  `json:"confirmedAt"`
	CreatedAt        time.Time  `json:"createdAt"`
	PropertyID       int        `json:"propertyId"`
	PropertyTitle    string     `json:"propertyTitle"`
	PropertyPhoto    *string    `json:"propertyPhoto"`
	PropertyAddress  string     `json:"propertyAddress"`
	PropertyCity     string     `json:"propertyCity"`
	PropertyDistrict string     `json:"propertyDistrict"`
	RequesterID      int        `json:"requesterId"`
	RequesterName    string     `json:"requesterName"`
	PropertyOwnerID  int        `json:"propertyOwnerId"`
	IsArchived       bool       `json:"isArchived"`
}

// Объект, который доступен пользователю для создания заявки на неисправность.
type AvailableRequestPropertyItem struct {
	ID       int     `json:"id"`
	Title    string  `json:"title"`
	Photo    *string `json:"photo"`
	Address  string  `json:"address"`
	City     string  `json:"city"`
	District string  `json:"district"`
}

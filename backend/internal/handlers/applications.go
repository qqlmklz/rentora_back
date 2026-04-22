package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"rentora/backend/internal/middleware"
	"rentora/backend/internal/models"
	"rentora/backend/internal/repository"
	"rentora/backend/internal/services"
	"rentora/backend/internal/utils"

	"github.com/gin-gonic/gin"
)

const requestExpenseUploadDir = "uploads/requests-expenses"
const requestUploadDir = "uploads/requests"

type createRequestPayload struct {
	PropertyID  json.RawMessage `json:"propertyId"`
	Title       json.RawMessage `json:"title"`
	Description json.RawMessage `json:"description"`
	Category    json.RawMessage `json:"category"`
}

type requestDecisionPayload struct {
	ResolutionType string `json:"resolutionType"`
}

func parseExpenseAmount(value string) (float64, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, errors.New("required")
	}
	amount, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, errors.New("format")
	}
	if amount < 0 {
		return 0, errors.New("format")
	}
	return amount, nil
}

func parsePropertyID(raw json.RawMessage) (int, error) {
	s := strings.TrimSpace(string(raw))
	if s == "" || s == "null" || s == `""` {
		return 0, errors.New("required")
	}
	var n int
	if err := json.Unmarshal(raw, &n); err == nil {
		return n, nil
	}
	var asString string
	if err := json.Unmarshal(raw, &asString); err != nil {
		return 0, errors.New("format")
	}
	asString = strings.TrimSpace(asString)
	if asString == "" {
		return 0, errors.New("required")
	}
	n, err := strconv.Atoi(asString)
	if err != nil {
		return 0, errors.New("format")
	}
	return n, nil
}

func parseRequiredText(raw json.RawMessage) (string, error) {
	s := strings.TrimSpace(string(raw))
	if s == "" || s == "null" || s == `""` {
		return "", errors.New("required")
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", errors.New("format")
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return "", errors.New("required")
	}
	return value, nil
}

func parseRequiredTextForm(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", errors.New("required")
	}
	return value, nil
}

func parsePropertyIDForm(value string) (int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, errors.New("required")
	}
	n, err := strconv.Atoi(value)
	if err != nil || n < 1 {
		return 0, errors.New("format")
	}
	return n, nil
}

func saveRequestPhotos(c *gin.Context, userID int) []string {
	form := c.Request.MultipartForm
	if form == nil {
		return []string{}
	}
	files := form.File["photos"]
	if len(files) == 0 {
		return []string{}
	}
	if err := os.MkdirAll(requestUploadDir, 0755); err != nil {
		log.Printf("[requests] create photos mkdir failed currentUserId=%d err=%v", userID, err)
		return []string{}
	}

	saved := make([]string, 0, len(files))
	for _, f := range files {
		ext := filepath.Ext(f.Filename)
		filename := fmt.Sprintf("%d_%d%s", userID, time.Now().UnixNano(), ext)
		dst := filepath.Join(requestUploadDir, filename)
		if err := c.SaveUploadedFile(f, dst); err != nil {
			log.Printf("[requests] create photo save failed currentUserId=%d file=%q err=%v", userID, f.Filename, err)
			continue
		}
		saved = append(saved, "/uploads/requests/"+filename)
	}
	return saved
}

// GetProfileRequests handles GET /api/profile/requests (activeRequests + archivedRequests — плоские массивы).
func GetProfileRequests(applicationService *services.ApplicationService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := middleware.GetUserID(c)
		if !ok {
			utils.JSONErrorUnauthorized(c, "Требуется авторизация")
			return
		}
		bucket := strings.ToLower(strings.TrimSpace(c.Query("bucket")))
		resp, err := applicationService.ListProfileRequests(c.Request.Context(), userID, bucket)
		if err != nil {
			log.Printf("[requests] GetProfileRequests: %v", err)
			utils.JSONErrorInternal(c, "Ошибка загрузки заявок")
			return
		}
		if resp == nil {
			resp = &models.ProfileRequestsResponse{}
		}
		if resp.ActiveRequests == nil {
			resp.ActiveRequests = []models.ProfileRequestsEntry{}
		}
		if resp.ArchivedRequests == nil {
			resp.ArchivedRequests = []models.ProfileRequestsEntry{}
		}
		c.JSON(http.StatusOK, resp)
	}
}

// CreateRequest handles POST /api/requests.
func CreateRequest(applicationService *services.ApplicationService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := middleware.GetUserID(c)
		if !ok {
			utils.JSONErrorUnauthorized(c, "Требуется авторизация")
			return
		}
		var (
			propertyID   int
			title        string
			description  string
			category     string
			requestPhotos []string
		)
		ct := c.ContentType()
		if strings.HasPrefix(ct, "multipart/") {
			if err := c.Request.ParseMultipartForm(32 << 20); err != nil {
				utils.JSONErrorBadRequest(c, "Ожидается корректный multipart/form-data")
				return
			}
			var parseErr error
			propertyID, parseErr = parsePropertyIDForm(c.PostForm("propertyId"))
			if parseErr != nil {
				switch parseErr.Error() {
				case "required":
					utils.JSONErrorBadRequest(c, "Поле propertyId обязательно")
				default:
					utils.JSONErrorBadRequest(c, "Поле propertyId имеет неверный формат")
				}
				return
			}
			title, parseErr = parseRequiredTextForm(c.PostForm("title"))
			if parseErr != nil {
				utils.JSONErrorBadRequest(c, "Поле title обязательно")
				return
			}
			description, parseErr = parseRequiredTextForm(c.PostForm("description"))
			if parseErr != nil {
				utils.JSONErrorBadRequest(c, "Поле description обязательно")
				return
			}
			category, parseErr = parseRequiredTextForm(c.PostForm("category"))
			if parseErr != nil {
				utils.JSONErrorBadRequest(c, "Поле category обязательно")
				return
			}
			requestPhotos = saveRequestPhotos(c, userID)
			log.Printf("[requests] create multipart method=%s path=%s currentUserId=%d propertyId=%d photosSaved=%d",
				c.Request.Method, c.Request.URL.Path, userID, propertyID, len(requestPhotos))
		} else {
			bodyBytes, err := c.GetRawData()
			if err != nil {
				utils.JSONErrorBadRequest(c, "Некорректный JSON")
				return
			}

			var payload createRequestPayload
			if err := json.Unmarshal(bodyBytes, &payload); err != nil {
				utils.JSONErrorBadRequest(c, "Некорректный JSON")
				return
			}
			log.Printf("[requests] create incoming method=%s path=%s currentUserId=%d body=%s propertyIdRaw=%s",
				c.Request.Method, c.Request.URL.Path, userID, string(bodyBytes), strings.TrimSpace(string(payload.PropertyID)))

			var parseErr error
			propertyID, parseErr = parsePropertyID(payload.PropertyID)
			if parseErr != nil {
				switch parseErr.Error() {
				case "required":
					utils.JSONErrorBadRequest(c, "Поле propertyId обязательно")
				default:
					utils.JSONErrorBadRequest(c, "Поле propertyId имеет неверный формат")
				}
				return
			}
			if propertyID < 1 {
				utils.JSONErrorBadRequest(c, "Поле propertyId имеет неверный формат")
				return
			}
			title, err = parseRequiredText(payload.Title)
			if err != nil {
				switch err.Error() {
				case "required":
					utils.JSONErrorBadRequest(c, "Поле title обязательно")
				default:
					utils.JSONErrorBadRequest(c, "Поле title имеет неверный формат")
				}
				return
			}
			description, err = parseRequiredText(payload.Description)
			if err != nil {
				switch err.Error() {
				case "required":
					utils.JSONErrorBadRequest(c, "Поле description обязательно")
				default:
					utils.JSONErrorBadRequest(c, "Поле description имеет неверный формат")
				}
				return
			}
			category, err = parseRequiredText(payload.Category)
			if err != nil {
				switch err.Error() {
				case "required":
					utils.JSONErrorBadRequest(c, "Поле category обязательно")
				default:
					utils.JSONErrorBadRequest(c, "Поле category имеет неверный формат")
				}
				return
			}
			requestPhotos = []string{}
		}

		req := models.CreateRequestBody{
			PropertyID:  &propertyID,
			Title:       title,
			Description: description,
			Category:    category,
		}
		bodyForLog, _ := json.Marshal(req)
		log.Printf("[requests] create method=%s path=%s currentUserId=%d requestBody=%s requestPhotosCount=%d",
			c.Request.Method, c.Request.URL.Path, userID, string(bodyForLog), len(requestPhotos))
		resp, err := applicationService.CreateRequest(c.Request.Context(), userID, req, requestPhotos)
		if err != nil {
			switch {
			case errors.Is(err, services.ErrInvalidApplicationInput):
				utils.JSONErrorBadRequest(c, "Некорректные данные заявки")
			case errors.Is(err, services.ErrRequestPropertyForbidden):
				utils.JSONErrorForbidden(c, "Нет доступа к этому объекту")
			case errors.Is(err, repository.ErrPropertyNotFound):
				utils.JSONErrorNotFound(c, "Объявление не найдено")
			default:
				log.Printf("[requests] CreateRequest: %v", err)
				utils.JSONErrorInternal(c, "Ошибка создания заявки")
			}
			return
		}
		c.JSON(http.StatusCreated, resp)
	}
}

// GetAvailableRequestProperties handles GET /api/requests/available-properties.
func GetAvailableRequestProperties(applicationService *services.ApplicationService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := middleware.GetUserID(c)
		if !ok {
			utils.JSONErrorUnauthorized(c, "Требуется авторизация")
			return
		}
		log.Println("[available-properties] userId:", userID)
		items, err := applicationService.ListAvailableProperties(c.Request.Context(), userID)
		if err != nil {
			log.Printf("[requests] GetAvailableRequestProperties: %v", err)
			utils.JSONErrorInternal(c, "Ошибка загрузки доступных объектов")
			return
		}
		if items == nil {
			items = []models.AvailableRequestPropertyItem{}
		}
		c.JSON(http.StatusOK, items)
	}
}

// DecideRequest handles PATCH /api/requests/:id/decision.
func DecideRequest(applicationService *services.ApplicationService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := middleware.GetUserID(c)
		if !ok {
			utils.JSONErrorUnauthorized(c, "Требуется авторизация")
			return
		}

		requestID, err := strconv.Atoi(strings.TrimSpace(c.Param("id")))
		if err != nil || requestID < 1 {
			utils.JSONErrorBadRequest(c, "Некорректный id заявки")
			return
		}

		var payload requestDecisionPayload
		if err := c.ShouldBindJSON(&payload); err != nil {
			utils.JSONErrorBadRequest(c, "Некорректный JSON")
			return
		}

		resp, err := applicationService.DecideRequest(c.Request.Context(), userID, requestID, payload.ResolutionType)
		if err != nil {
			switch {
			case errors.Is(err, services.ErrRequestDecisionInvalidResolution):
				utils.JSONErrorBadRequest(c, "Поле resolutionType должно быть owner или tenant")
			case errors.Is(err, services.ErrRequestDecisionInvalidStatus):
				utils.JSONErrorBadRequest(c, "Решение можно принять только для заявок в статусе pending или in_review")
			case errors.Is(err, services.ErrRequestDecisionForbidden):
				utils.JSONErrorForbidden(c, "Нет доступа к этой заявке")
			case errors.Is(err, services.ErrRequestDecisionNotFound):
				utils.JSONErrorNotFound(c, "Заявка не найдена")
			default:
				log.Printf("[requests] DecideRequest: requestId=%d currentUserId=%d err=%v", requestID, userID, err)
				utils.JSONErrorInternal(c, "Ошибка принятия решения по заявке")
			}
			return
		}

		c.JSON(http.StatusOK, resp)
	}
}

// SubmitRequestExpense handles PATCH /api/requests/:id/expense.
func SubmitRequestExpense(applicationService *services.ApplicationService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := middleware.GetUserID(c)
		if !ok {
			utils.JSONErrorUnauthorized(c, "Требуется авторизация")
			return
		}

		requestID, err := strconv.Atoi(strings.TrimSpace(c.Param("id")))
		if err != nil || requestID < 1 {
			utils.JSONErrorBadRequest(c, "Некорректный id заявки")
			return
		}

		if err := c.Request.ParseMultipartForm(32 << 20); err != nil {
			utils.JSONErrorBadRequest(c, "Ожидается multipart/form-data")
			return
		}

		expenseAmount, err := parseExpenseAmount(c.PostForm("expenseAmount"))
		if err != nil {
			switch err.Error() {
			case "required":
				utils.JSONErrorBadRequest(c, "Поле expenseAmount обязательно")
			default:
				utils.JSONErrorBadRequest(c, "Поле expenseAmount имеет неверный формат")
			}
			return
		}
		expenseComment := strings.TrimSpace(c.PostForm("expenseComment"))
		if expenseComment == "" {
			utils.JSONErrorBadRequest(c, "Поле expenseComment обязательно")
			return
		}

		var expensePhotos []string
		if c.Request.MultipartForm != nil {
			files := c.Request.MultipartForm.File["expensePhotos"]
			if len(files) > 0 {
				if err := os.MkdirAll(requestExpenseUploadDir, 0755); err != nil {
					log.Printf("[requests-expense] mkdir failed requestId=%d currentUserId=%d err=%v", requestID, userID, err)
					utils.JSONErrorInternal(c, "Ошибка сохранения фото затрат")
					return
				}
				expensePhotos = make([]string, 0, len(files))
				for _, f := range files {
					ext := filepath.Ext(f.Filename)
					filename := fmt.Sprintf("%d_%d_%d%s", userID, requestID, time.Now().UnixNano(), ext)
					dst := filepath.Join(requestExpenseUploadDir, filename)
					if err := c.SaveUploadedFile(f, dst); err != nil {
						log.Printf("[requests-expense] save photo failed requestId=%d currentUserId=%d file=%q err=%v", requestID, userID, f.Filename, err)
						utils.JSONErrorInternal(c, "Ошибка сохранения фото затрат")
						return
					}
					expensePhotos = append(expensePhotos, "/uploads/requests-expenses/"+filename)
				}
			}
		}
		if expensePhotos == nil {
			expensePhotos = []string{}
		}

		resp, err := applicationService.SubmitRequestExpense(c.Request.Context(), userID, requestID, expenseAmount, expenseComment, expensePhotos)
		if err != nil {
			switch {
			case errors.Is(err, services.ErrRequestExpenseInvalidAmount):
				utils.JSONErrorBadRequest(c, "Поле expenseAmount имеет неверный формат")
			case errors.Is(err, services.ErrRequestExpenseInvalidScenario):
				utils.JSONErrorBadRequest(c, "Затраты можно отправить только в сценарии tenant_resolves")
			case errors.Is(err, services.ErrRequestExpenseForbidden):
				utils.JSONErrorForbidden(c, "Нет доступа к этой заявке")
			case errors.Is(err, services.ErrRequestDecisionNotFound):
				utils.JSONErrorNotFound(c, "Заявка не найдена")
			default:
				log.Printf("[requests-expense] submit failed requestId=%d currentUserId=%d err=%v", requestID, userID, err)
				utils.JSONErrorInternal(c, "Ошибка отправки затрат по заявке")
			}
			return
		}
		c.JSON(http.StatusOK, resp)
	}
}

// ConfirmTenantExpenses handles POST /api/requests/:id/confirm-tenant-expenses.
func ConfirmTenantExpenses(applicationService *services.ApplicationService) gin.HandlerFunc {
	return func(c *gin.Context) {
		ownerID, ok := middleware.GetUserID(c)
		if !ok {
			utils.JSONErrorUnauthorized(c, "Требуется авторизация")
			return
		}
		requestID, err := strconv.Atoi(strings.TrimSpace(c.Param("id")))
		if err != nil || requestID < 1 {
			utils.JSONErrorBadRequest(c, "Некорректный id заявки")
			return
		}
		resp, err := applicationService.ConfirmTenantExpenses(c.Request.Context(), ownerID, requestID)
		if err != nil {
			switch {
			case errors.Is(err, services.ErrConfirmTenantExpensesForbidden):
				utils.JSONErrorForbidden(c, "Подтвердить расходы может только владелец объекта")
			case errors.Is(err, services.ErrConfirmTenantExpensesNotFound):
				utils.JSONErrorNotFound(c, "Заявка не найдена")
			case errors.Is(err, services.ErrConfirmTenantExpensesInvalidResolution):
				utils.JSONErrorBadRequest(c, "Подтверждение доступно только для сценария resolution_type=tenant")
			case errors.Is(err, services.ErrConfirmTenantExpensesNoExpenses):
				utils.JSONErrorBadRequest(c, "Нет данных о расходах для подтверждения")
			case errors.Is(err, services.ErrConfirmTenantExpensesWrongStatus):
				utils.JSONErrorBadRequest(c, "Заявка не в статусе ожидания подтверждения расходов")
			case errors.Is(err, services.ErrConfirmTenantExpensesAlready):
				utils.JSONErrorBadRequest(c, "Расходы по этой заявке уже подтверждены")
			default:
				log.Printf("[requests-confirm-expenses] requestId=%d ownerUserId=%d err=%v", requestID, ownerID, err)
				utils.JSONErrorInternal(c, "Ошибка подтверждения расходов")
			}
			return
		}
		c.JSON(http.StatusOK, resp)
	}
}

// CompleteOwnerResolution handles POST /api/requests/:id/complete-owner.
func CompleteOwnerResolution(applicationService *services.ApplicationService) gin.HandlerFunc {
	return func(c *gin.Context) {
		ownerID, ok := middleware.GetUserID(c)
		if !ok {
			utils.JSONErrorUnauthorized(c, "Требуется авторизация")
			return
		}
		requestID, err := strconv.Atoi(strings.TrimSpace(c.Param("id")))
		if err != nil || requestID < 1 {
			utils.JSONErrorBadRequest(c, "Некорректный id заявки")
			return
		}
		resp, err := applicationService.CompleteOwnerResolution(c.Request.Context(), ownerID, requestID)
		if err != nil {
			switch {
			case errors.Is(err, services.ErrCompleteOwnerNotFound):
				utils.JSONErrorNotFound(c, "Заявка не найдена")
			case errors.Is(err, services.ErrCompleteOwnerForbidden):
				utils.JSONErrorForbidden(c, "Завершить заявку может только владелец объекта")
			case errors.Is(err, services.ErrCompleteOwnerInvalidResolution):
				utils.JSONErrorBadRequest(c, "Завершение доступно только для сценария resolution_type=owner")
			case errors.Is(err, services.ErrCompleteOwnerWrongStatus):
				utils.JSONErrorBadRequest(c, "Заявка не в статусе ожидания действий владельца")
			case errors.Is(err, services.ErrCompleteOwnerAlreadyDone):
				utils.JSONErrorBadRequest(c, "Заявка уже завершена")
			default:
				log.Printf("[requests-complete-owner] requestId=%d ownerUserId=%d err=%v", requestID, ownerID, err)
				utils.JSONErrorInternal(c, "Ошибка завершения заявки")
			}
			return
		}
		c.JSON(http.StatusOK, resp)
	}
}

// CompleteOwnerRequest handles POST /api/requests/:id/complete-owner-request (owner flow → status completed).
func CompleteOwnerRequest(applicationService *services.ApplicationService) gin.HandlerFunc {
	return func(c *gin.Context) {
		ownerID, ok := middleware.GetUserID(c)
		if !ok {
			utils.JSONErrorUnauthorized(c, "Требуется авторизация")
			return
		}
		requestID, err := strconv.Atoi(strings.TrimSpace(c.Param("id")))
		if err != nil || requestID < 1 {
			utils.JSONErrorBadRequest(c, "Некорректный id заявки")
			return
		}
		resp, err := applicationService.CompleteOwnerRequest(c.Request.Context(), ownerID, requestID)
		if err != nil {
			switch {
			case errors.Is(err, services.ErrCompleteOwnerNotFound):
				utils.JSONErrorNotFound(c, "Заявка не найдена")
			case errors.Is(err, services.ErrCompleteOwnerForbidden):
				utils.JSONErrorForbidden(c, "Завершить заявку может только владелец объекта")
			case errors.Is(err, services.ErrCompleteOwnerInvalidResolution):
				utils.JSONErrorBadRequest(c, "Завершение доступно только для сценария resolution_type=owner")
			case errors.Is(err, services.ErrCompleteOwnerWrongStatus):
				utils.JSONErrorBadRequest(c, "Заявка не в статусе ожидания действий владельца")
			case errors.Is(err, services.ErrCompleteOwnerAlreadyDone):
				utils.JSONErrorBadRequest(c, "Заявка уже завершена")
			default:
				log.Printf("[requests-complete-owner-request] requestId=%d ownerUserId=%d err=%v", requestID, ownerID, err)
				utils.JSONErrorInternal(c, "Ошибка завершения заявки")
			}
			return
		}
		c.JSON(http.StatusOK, resp)
	}
}

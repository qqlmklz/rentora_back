package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"rentora/backend/internal/middleware"
	"rentora/backend/internal/models"
	"rentora/backend/internal/services"
	"rentora/backend/internal/utils"

	"github.com/gin-gonic/gin"
)

const (
	propertyPhotosKey       = "photos"
	existingPhotosFormKey   = "existingPhotos"
	minPropertyPhotos       = 5
	propertyUploadDir       = "uploads/properties"
)

// Обработчик GET /api/properties для каталога.
func GetProperties(propertyService *services.PropertyService) gin.HandlerFunc {
	return func(c *gin.Context) {
		filters := services.CatalogFilters{
			Category:     c.Query("category"),
			PropertyType: c.Query("propertyType"),
			Location:     c.Query("location"),
			Sort:         c.DefaultQuery("sort", "newest"),
		}

		if roomsStr := c.Query("rooms"); roomsStr != "" {
			if v, err := strconv.Atoi(roomsStr); err == nil {
				filters.Rooms = v
			}
		}
		if priceFromStr := c.Query("priceFrom"); priceFromStr != "" {
			if v, err := strconv.Atoi(priceFromStr); err == nil {
				filters.PriceFrom = v
			}
		}
		if priceToStr := c.Query("priceTo"); priceToStr != "" {
			if v, err := strconv.Atoi(priceToStr); err == nil {
				filters.PriceTo = v
			}
		}

		props, err := propertyService.ListForCatalog(c.Request.Context(), filters)
		if err != nil {
			// Для каталога пока отдаем общий 500; если нужно, потом добавим отдельное логирование.
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "Ошибка загрузки объявлений"})
			return
		}
		c.JSON(http.StatusOK, props)
	}
}

// Обработчик GET /api/properties/:id (публичный). С Bearer JWT владелец видит apartmentNumber.
func GetPropertyByID(propertyService *services.PropertyService, jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil || id < 1 {
			utils.JSONErrorBadRequest(c, "Некорректный id")
			return
		}
		detail, err := propertyService.GetByID(c.Request.Context(), id)
		if err != nil {
			if errors.Is(err, services.ErrPropertyNotFound) {
				utils.JSONErrorNotFound(c, "Объявление не найдено")
				return
			}
			log.Printf("[properties] GetByID: %v", err)
			utils.JSONErrorInternal(c, "Ошибка загрузки объявления")
			return
		}
		uid, ok := middleware.ParseUserIDFromBearer(c, jwtSecret)
		if !ok || detail.OwnerID == nil || *detail.OwnerID != uid {
			detail.ApartmentNumber = nil
		}
		c.JSON(http.StatusOK, detail)
	}
}

// Обработчик POST /api/properties (multipart/form-data).
func CreateProperty(propertyService *services.PropertyService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := middleware.GetUserID(c)
		if !ok {
			utils.JSONErrorUnauthorized(c, "Требуется авторизация")
			return
		}

		// Сначала парсим обязательные базовые поля.
		in, err := parseCreatePropertyInput(c)
		if err != nil {
			utils.JSONErrorBadRequest(c, err.Error())
			return
		}

		form, err := c.MultipartForm()
		if err != nil {
			utils.JSONErrorBadRequest(c, "Ожидается multipart/form-data")
			return
		}
		files := form.File[propertyPhotosKey]
		if len(files) < minPropertyPhotos {
			utils.JSONErrorBadRequest(c, "Нужно загрузить минимум 5 фото")
			return
		}

		if err := os.MkdirAll(propertyUploadDir, 0755); err != nil {
			log.Printf("[properties] mkdir: %v", err)
			utils.JSONErrorInternal(c, "Ошибка сохранения фото")
			return
		}

		var urls []string
		for _, f := range files {
			ext := filepath.Ext(f.Filename)
			name := fmt.Sprintf("%d_%d%s", userID, time.Now().UnixNano(), ext)
			dst := filepath.Join(propertyUploadDir, name)
			if err := c.SaveUploadedFile(f, dst); err != nil {
				log.Printf("[properties] save photo: %v", err)
				utils.JSONErrorInternal(c, "Ошибка сохранения фото")
				return
			}
			urls = append(urls, "/uploads/properties/"+name)
		}

		id, err := propertyService.Create(c.Request.Context(), userID, in, urls)
		if err != nil {
			if err == services.ErrInvalidCategoryPropertyType {
				utils.JSONErrorBadRequest(c, "propertyType не соответствует category")
				return
			}
			log.Printf("[properties] create: %v", err)
			utils.JSONErrorInternal(c, "Ошибка создания объявления")
			return
		}

		c.JSON(http.StatusCreated, models.PropertyCreateResponse{
			ID:       id,
			Title:    in.Title,
			City:     in.City,
			District: in.District,
			Images:   urls,
		})
	}
}

// Обработчик GET /api/profile/properties (JWT).
func GetMyProperties(propertyService *services.PropertyService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := middleware.GetUserID(c)
		if !ok {
			utils.JSONErrorUnauthorized(c, "Требуется авторизация")
			return
		}
		props, err := propertyService.ListMine(c.Request.Context(), userID)
		if err != nil {
			log.Printf("[properties] ListMine: %v", err)
			utils.JSONErrorInternal(c, "Ошибка загрузки объявлений")
			return
		}
		c.JSON(http.StatusOK, props)
	}
}

// Обработчик DELETE /api/properties/:id (JWT, только владелец).
func DeleteProperty(propertyService *services.PropertyService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := middleware.GetUserID(c)
		if !ok {
			utils.JSONErrorUnauthorized(c, "Требуется авторизация")
			return
		}
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil || id < 1 {
			utils.JSONErrorBadRequest(c, "Некорректный id")
			return
		}
		err = propertyService.DeleteOwned(c.Request.Context(), userID, id)
		if err != nil {
			switch {
			case errors.Is(err, services.ErrPropertyNotFound):
				utils.JSONErrorNotFound(c, "Объявление не найдено")
			case errors.Is(err, services.ErrPropertyForbidden):
				utils.JSONErrorForbidden(c, "Нет доступа к этому объявлению")
			default:
				log.Printf("[properties] Delete: %v", err)
				utils.JSONErrorInternal(c, "Ошибка удаления объявления")
			}
			return
		}
		c.Status(http.StatusNoContent)
	}
}

// Обработчик PATCH /api/properties/:id (JSON или multipart: payload JSON + existingPhotos + файлы photos).
func UpdateProperty(propertyService *services.PropertyService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := middleware.GetUserID(c)
		if !ok {
			utils.JSONErrorUnauthorized(c, "Требуется авторизация")
			return
		}
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil || id < 1 {
			utils.JSONErrorBadRequest(c, "Некорректный id")
			return
		}

		var payload models.UpdatePropertyPayload
		var newFiles []*multipart.FileHeader

		ct := c.ContentType()
		if strings.HasPrefix(ct, "multipart/") {
			if err := c.Request.ParseMultipartForm(32 << 20); err != nil {
				utils.JSONErrorBadRequest(c, "Некорректный multipart")
				return
			}
			payloadStr := c.PostForm("payload")
			if payloadStr == "" {
				payloadStr = "{}"
			}
			if err := json.Unmarshal([]byte(payloadStr), &payload); err != nil {
				utils.JSONErrorBadRequest(c, "Поле payload должно быть валидным JSON")
				return
			}
			if c.Request.MultipartForm != nil && c.Request.MultipartForm.File != nil {
				newFiles = c.Request.MultipartForm.File[propertyPhotosKey]
			}
			// Важно: existingPhotos — это JSON-массив строк в form-data, по нему синхронизируем старые фото.
			existingPhotosRaw := strings.TrimSpace(c.PostForm(existingPhotosFormKey))
			if existingPhotosRaw != "" {
				var parsed []string
				if err := json.Unmarshal([]byte(existingPhotosRaw), &parsed); err != nil {
					utils.JSONErrorBadRequest(c, "Поле existingPhotos имеет неверный формат")
					return
				}
				payload.ExistingPhotos = &parsed
				log.Printf("[properties] PATCH multipart: existingPhotosRaw=%q parsedExistingPhotos=%v newFilesCount=%d",
					existingPhotosRaw, parsed, len(newFiles))
			}
		} else {
			if err := c.ShouldBindJSON(&payload); err != nil {
				utils.JSONErrorBadRequest(c, "Некорректный JSON")
				return
			}
		}

		if !payload.HasMetaChanges() && len(newFiles) == 0 {
			utils.JSONErrorBadRequest(c, "Нет данных для обновления")
			return
		}

		var newURLs []string
		if len(newFiles) > 0 {
			if err := os.MkdirAll(propertyUploadDir, 0755); err != nil {
				log.Printf("[properties] mkdir: %v", err)
				utils.JSONErrorInternal(c, "Ошибка сохранения фото")
				return
			}
			for _, f := range newFiles {
				ext := filepath.Ext(f.Filename)
				name := fmt.Sprintf("%d_%d%s", userID, time.Now().UnixNano(), ext)
				dst := filepath.Join(propertyUploadDir, name)
				if err := c.SaveUploadedFile(f, dst); err != nil {
					log.Printf("[properties] save photo: %v", err)
					utils.JSONErrorInternal(c, "Ошибка сохранения фото")
					return
				}
				newURLs = append(newURLs, "/uploads/properties/"+name)
			}
		}

		photos, err := propertyService.UpdateOwned(c.Request.Context(), userID, id, payload, newURLs)
		if err != nil {
			switch {
			case errors.Is(err, services.ErrEmptyPropertyPatch):
				utils.JSONErrorBadRequest(c, "Нет данных для обновления")
			case errors.Is(err, services.ErrPropertyNotFound):
				utils.JSONErrorNotFound(c, "Объявление не найдено")
			case errors.Is(err, services.ErrPropertyForbidden):
				utils.JSONErrorForbidden(c, "Нет доступа к этому объявлению")
			case errors.Is(err, services.ErrInvalidCategoryPropertyType):
				utils.JSONErrorBadRequest(c, "propertyType не соответствует category")
			default:
				log.Printf("[properties] Update: %v", err)
				utils.JSONErrorInternal(c, "Ошибка обновления объявления")
			}
			return
		}
		c.JSON(http.StatusOK, gin.H{"photos": photos})
	}
}

func parseCreatePropertyInput(c *gin.Context) (models.CreatePropertyInput, error) {
	get := func(key string) string { return c.PostForm(key) }

	// Маппинг входных значений.

	// Поле rentType: long -> долгосрочная, daily -> посуточная.
	mapRentType := func(v string) string {
		switch v {
		case "long":
			return "долгосрочная"
		case "daily":
			return "посуточная"
		default:
			return v
		}
	}

	// Поле category: residential -> жилая, commercial -> коммерческая.
	mapCategory := func(v string) string {
		switch v {
		case "residential":
			return "жилая"
		case "commercial":
			return "коммерческая"
		default:
			return v
		}
	}

	// Поле subcategory переводим в propertyType.
	mapPropertyType := func(v string) string {
		switch v {
		case "apartment":
			return "квартира"
		case "room":
			return "комната"
		case "house":
			return "дом/дача"
		case "cottage":
			return "коттедж"
		case "office":
			return "офис"
		case "coworking":
			return "коворкинг"
		case "building":
			return "здание"
		case "warehouse":
			return "склад"
		default:
			return v
		}
	}

	// Поле residentialType переводим в housingType.
	mapHousingType := func(v string) string {
		switch v {
		case "flat":
			return "квартира"
		case "apartments":
			return "апартаменты"
		default:
			return v
		}
	}

	// Поле prepayment: 0 -> нет, 1 -> 1 месяц, 2 -> 2 месяца.
	mapPrepayment := func(v string) string {
		switch v {
		case "0":
			return "нет"
		case "1":
			return "1 месяц"
		case "2":
			return "2 месяца"
		default:
			return v
		}
	}

	// Читаем поля и приводим к нужному формату.

	categoryRaw := get("category")
	category := mapCategory(categoryRaw)
	isCommercial := category == "коммерческая"

	req := models.CreatePropertyInput{
		Title:        get("title"),
		RentType:     mapRentType(get("rentType")),
		Category:     category,
		PropertyType: mapPropertyType(get("subcategory")),
		Address:      get("address"),
		City:         get("city"),
		District:     get("district"),
	}

	// Проверяем, что обязательные поля реально пришли.

	var missing []string
	if get("title") == "" {
		missing = append(missing, "title")
	}
	if get("rentType") == "" {
		missing = append(missing, "rentType")
	}
	if get("category") == "" {
		missing = append(missing, "category")
	}
	if get("subcategory") == "" {
		missing = append(missing, "subcategory")
	}
	if get("address") == "" {
		missing = append(missing, "address")
	}
	if get("city") == "" {
		missing = append(missing, "city")
	}
	if get("district") == "" {
		missing = append(missing, "district")
	}
	if get("price") == "" {
		missing = append(missing, "price")
	}
	if get("utilitiesIncluded") == "" {
		missing = append(missing, "utilitiesIncluded")
	}
	if get("totalArea") == "" {
		missing = append(missing, "totalArea")
	}
	if len(missing) > 0 {
		return models.CreatePropertyInput{}, fmt.Errorf("Не заполнены обязательные поля: %s", strings.Join(missing, ", "))
	}

	// Локальные вспомогательные функции для парсинга.

	// Парсим bool: true/false/included/not_included.
	parseBool := func(key string) (bool, error) {
		s := get(key)
		if s == "" {
			return false, nil
		}
		switch s {
		case "true", "1", "on", "yes", "included":
			return true, nil
		case "false", "0", "off", "no", "not_included":
			return false, nil
		default:
			return false, fmt.Errorf("Поле %s имеет неверный формат", key)
		}
	}

	// Парсим int.
	atoi := func(key string) (int, bool, error) {
		s := get(key)
		if s == "" {
			return 0, false, nil
		}
		i, err := strconv.Atoi(s)
		if err != nil {
			return 0, false, fmt.Errorf("Поле %s имеет неверный формат", key)
		}
		return i, true, nil
	}

	// Парсим float.
	atof := func(key string) (float64, bool, error) {
		s := get(key)
		if s == "" {
			return 0, false, nil
		}
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return 0, false, fmt.Errorf("Поле %s имеет неверный формат", key)
		}
		return f, true, nil
	}

	// Разбираем числовые поля.

	// Поле цены (обязательное).
	price, _, err := atoi("price")
	if err != nil {
		return models.CreatePropertyInput{}, err
	}
	req.Price = price

	// Поле totalArea (обязательное).
	totalArea, _, err := atof("totalArea")
	if err != nil {
		return models.CreatePropertyInput{}, err
	}
	req.TotalArea = totalArea

	// Поле rooms: studio -> 0, 6+ -> 6, иначе обычное число.
	// Для коммерческой недвижимости rooms не обязателен.
	roomsStr := get("rooms")
	if roomsStr != "" {
		switch roomsStr {
		case "studio":
			req.Rooms = 0
		case "6+":
			req.Rooms = 6
		default:
			rooms, err := strconv.Atoi(roomsStr)
			if err != nil {
				return models.CreatePropertyInput{}, fmt.Errorf("Поле rooms имеет неверный формат")
			}
			req.Rooms = rooms
		}
	} else if !isCommercial {
		// Для жилой недвижимости rooms обязателен, но в missing не дублируем — уже проверили выше.
	}

	// Разбираем bool-поля.

	// Поле utilitiesIncluded (обязательное).
	utilsIncluded, err := parseBool("utilitiesIncluded")
	if err != nil {
		return models.CreatePropertyInput{}, err
	}
	req.UtilitiesIncluded = utilsIncluded

	// Поле utilitiesPrice обязательно, если utilitiesIncluded = false (not_included).
	if !utilsIncluded {
		if get("utilitiesPrice") == "" {
			return models.CreatePropertyInput{}, fmt.Errorf("Поле utilitiesPrice обязательно, если utilitiesIncluded = not_included")
		}
	}
	if v, has, err := atoi("utilitiesPrice"); err != nil {
		return models.CreatePropertyInput{}, err
	} else if has {
		req.UtilitiesPrice = &v
	}

	// Поле allowChildren переводим в childrenAllowed.
	childrenAllowed, err := parseBool("allowChildren")
	if err != nil {
		return models.CreatePropertyInput{}, err
	}
	req.ChildrenAllowed = childrenAllowed

	// Поле allowPets переводим в petsAllowed.
	petsAllowed, err := parseBool("allowPets")
	if err != nil {
		return models.CreatePropertyInput{}, err
	}
	req.PetsAllowed = petsAllowed

	// Разбираем необязательные поля.

	// поле metro.
	if v := get("metro"); v != "" {
		req.Metro = &v
	}

	// поле apartmentNumber.
	if v := get("apartmentNumber"); v != "" {
		req.ApartmentNumber = &v
	}

	// Поле residentialType переводим в housingType (для коммерческой не обязательно).
	if v := get("residentialType"); v != "" {
		ht := mapHousingType(v)
		req.HousingType = &ht
	}

	// поле deposit.
	if v, has, err := atoi("deposit"); err != nil {
		return models.CreatePropertyInput{}, err
	} else if has {
		req.Deposit = &v
	}

	// поле commission перекладываем в commissionPercent.
	if v, has, err := atoi("commission"); err != nil {
		return models.CreatePropertyInput{}, err
	} else if has {
		req.CommissionPercent = &v
	}

	// поле prepayment.
	if v := get("prepayment"); v != "" {
		p := mapPrepayment(v)
		req.Prepayment = &p
	}

	// поле livingArea.
	if v, has, err := atof("livingArea"); err != nil {
		return models.CreatePropertyInput{}, err
	} else if has {
		req.LivingArea = &v
	}

	// поле kitchenArea.
	if v, has, err := atof("kitchenArea"); err != nil {
		return models.CreatePropertyInput{}, err
	} else if has {
		req.KitchenArea = &v
	}

	// поле floor.
	if v, has, err := atoi("floor"); err != nil {
		return models.CreatePropertyInput{}, err
	} else if has {
		req.Floor = &v
	}

	// поле floorsTotal перекладываем в totalFloors.
	if v, has, err := atoi("floorsTotal"); err != nil {
		return models.CreatePropertyInput{}, err
	} else if has {
		req.TotalFloors = &v
	}

	return req, nil
}


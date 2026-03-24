package handlers

import (
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
	"rentora/backend/internal/services"
	"rentora/backend/internal/utils"

	"github.com/gin-gonic/gin"
)

const (
	propertyPhotosKey   = "photos"
	minPropertyPhotos   = 5
	propertyUploadDir   = "uploads/properties"
)

// GetProperties handles GET /api/properties for catalog.
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
			// For catalog we just return generic 500; logging can be added later.
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "Ошибка загрузки объявлений"})
			return
		}
		c.JSON(http.StatusOK, props)
	}
}

// GetPropertyByID handles GET /api/properties/:id (public, no auth).
func GetPropertyByID(propertyService *services.PropertyService) gin.HandlerFunc {
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
		c.JSON(http.StatusOK, detail)
	}
}

// CreateProperty handles POST /api/properties (multipart/form-data).
func CreateProperty(propertyService *services.PropertyService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := middleware.GetUserID(c)
		if !ok {
			utils.JSONErrorUnauthorized(c, "Требуется авторизация")
			return
		}

		// Parse required basic fields.
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

func parseCreatePropertyInput(c *gin.Context) (models.CreatePropertyInput, error) {
	get := func(key string) string { return c.PostForm(key) }

	// ===== МАППИНГ ЗНАЧЕНИЙ =====

	// rentType: long -> долгосрочная, daily -> посуточная
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

	// category: residential -> жилая, commercial -> коммерческая
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

	// subcategory -> propertyType
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

	// residentialType -> housingType
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

	// prepayment: 0 -> нет, 1 -> 1 месяц, 2 -> 2 месяца
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

	// ===== ЧТЕНИЕ И ПРЕОБРАЗОВАНИЕ ПОЛЕЙ =====

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

	// ===== ПРОВЕРКА ОБЯЗАТЕЛЬНЫХ ПОЛЕЙ =====

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

	// ===== ВСПОМОГАТЕЛЬНЫЕ ФУНКЦИИ =====

	// Парсинг bool: true/false/included/not_included
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

	// Парсинг int
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

	// Парсинг float
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

	// ===== ПАРСИНГ ЧИСЛОВЫХ ПОЛЕЙ =====

	// price (обязательное)
	price, _, err := atoi("price")
	if err != nil {
		return models.CreatePropertyInput{}, err
	}
	req.Price = price

	// totalArea (обязательное)
	totalArea, _, err := atof("totalArea")
	if err != nil {
		return models.CreatePropertyInput{}, err
	}
	req.TotalArea = totalArea

	// rooms: studio -> 0, 6+ -> 6, иначе число
	// Для коммерческой недвижимости не обязателен
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
		// rooms обязателен для жилой недвижимости — но не добавляем в missing, т.к. уже проверили выше
	}

	// ===== ПАРСИНГ BOOL ПОЛЕЙ =====

	// utilitiesIncluded (обязательное)
	utilsIncluded, err := parseBool("utilitiesIncluded")
	if err != nil {
		return models.CreatePropertyInput{}, err
	}
	req.UtilitiesIncluded = utilsIncluded

	// utilitiesPrice: обязателен, если utilitiesIncluded = false (not_included)
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

	// allowChildren -> childrenAllowed
	childrenAllowed, err := parseBool("allowChildren")
	if err != nil {
		return models.CreatePropertyInput{}, err
	}
	req.ChildrenAllowed = childrenAllowed

	// allowPets -> petsAllowed
	petsAllowed, err := parseBool("allowPets")
	if err != nil {
		return models.CreatePropertyInput{}, err
	}
	req.PetsAllowed = petsAllowed

	// ===== ОПЦИОНАЛЬНЫЕ ПОЛЯ =====

	// metro
	if v := get("metro"); v != "" {
		req.Metro = &v
	}

	// apartmentNumber
	if v := get("apartmentNumber"); v != "" {
		req.ApartmentNumber = &v
	}

	// residentialType -> housingType (не обязательно для коммерческой)
	if v := get("residentialType"); v != "" {
		ht := mapHousingType(v)
		req.HousingType = &ht
	}

	// deposit
	if v, has, err := atoi("deposit"); err != nil {
		return models.CreatePropertyInput{}, err
	} else if has {
		req.Deposit = &v
	}

	// commission -> commissionPercent
	if v, has, err := atoi("commission"); err != nil {
		return models.CreatePropertyInput{}, err
	} else if has {
		req.CommissionPercent = &v
	}

	// prepayment
	if v := get("prepayment"); v != "" {
		p := mapPrepayment(v)
		req.Prepayment = &p
	}

	// livingArea
	if v, has, err := atof("livingArea"); err != nil {
		return models.CreatePropertyInput{}, err
	} else if has {
		req.LivingArea = &v
	}

	// kitchenArea
	if v, has, err := atof("kitchenArea"); err != nil {
		return models.CreatePropertyInput{}, err
	} else if has {
		req.KitchenArea = &v
	}

	// floor
	if v, has, err := atoi("floor"); err != nil {
		return models.CreatePropertyInput{}, err
	} else if has {
		req.Floor = &v
	}

	// floorsTotal -> totalFloors
	if v, has, err := atoi("floorsTotal"); err != nil {
		return models.CreatePropertyInput{}, err
	} else if has {
		req.TotalFloors = &v
	}

	return req, nil
}


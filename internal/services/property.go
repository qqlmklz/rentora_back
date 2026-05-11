package services

import (
	"context"
	"errors"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"rentora/backend/internal/models"
	"rentora/backend/internal/repository"
)

// Сервис для объявлений: список, детали и CRUD-операции.
type PropertyService struct {
	repo *repository.DB
}

// Конструктор PropertyService.
func NewPropertyService(repo *repository.DB) *PropertyService {
	return &PropertyService{repo: repo}
}

var (
	ErrInvalidCategoryPropertyType = errors.New("invalid category/propertyType combination")
	ErrEmptyPropertyPatch          = errors.New("empty property patch")
)

// Фильтры, которые приходят для каталога.
type CatalogFilters struct {
	Category      string
	PropertyType  string
	RoomsExact    *int
	RoomsMin      *int
	PriceFrom     *int
	PriceTo       *int
	Location      string
	Sort          string
	CurrentUserID *int
}

const (
	recommendationsLimit = 6
)

// Возвращаем объявления для /catalog с примененными фильтрами и сортировкой.
func (s *PropertyService) ListForCatalog(ctx context.Context, f CatalogFilters) ([]models.Property, error) {
	log.Printf("[properties] catalog service filters: category=%q propertyType=%q roomsExact=%v roomsMin=%v priceFrom=%v priceTo=%v location=%q sort=%q",
		f.Category, f.PropertyType, f.RoomsExact, f.RoomsMin, f.PriceFrom, f.PriceTo, f.Location, f.Sort)
	return s.repo.ListProperties(ctx, repository.PropertyFilters{
		Category:      f.Category,
		PropertyType:  f.PropertyType,
		RoomsExact:    f.RoomsExact,
		RoomsMin:      f.RoomsMin,
		PriceFrom:     f.PriceFrom,
		PriceTo:       f.PriceTo,
		Location:      f.Location,
		Sort:          f.Sort,
		CurrentUserID: f.CurrentUserID,
	})
}

// Создаем объявление и сохраняем URL фото. Заодно проверяем связку category и propertyType.
func (s *PropertyService) Create(ctx context.Context, userID int, in models.CreatePropertyInput, imageURLs []string) (int, error) {
	if !isPropertyTypeAllowed(in.Category, in.PropertyType) {
		return 0, ErrInvalidCategoryPropertyType
	}
	return s.repo.CreatePropertyWithImages(ctx, userID, in, imageURLs)
}

// Возвращаем одно объявление для публичной страницы деталей.
func (s *PropertyService) GetByID(ctx context.Context, id int) (*models.PropertyDetail, error) {
	return s.repo.GetPropertyByID(ctx, id)
}

// Сохраняем просмотр объявления для последующих рекомендаций.
func (s *PropertyService) TrackView(ctx context.Context, userID, propertyID int) error {
	return s.repo.UpsertPropertyView(ctx, userID, propertyID)
}

// Возвращаем рекомендации по последним просмотренным объявлениям пользователя.
func (s *PropertyService) GetRecommendations(ctx context.Context, userID int) ([]models.Property, error) {
	return s.repo.ListRecommendations(ctx, userID, recommendationsLimit)
}

// Возвращаем объявления пользователя в формате карточек.
func (s *PropertyService) ListMine(ctx context.Context, userID int) ([]models.Property, error) {
	return s.repo.ListPropertiesByUserID(ctx, userID)
}

// Удаляем объявление только если оно принадлежит userID.
func (s *PropertyService) DeleteOwned(ctx context.Context, userID, propertyID int) error {
	return s.repo.DeletePropertyOwned(ctx, propertyID, userID)
}

// Применяем payload и синхронизируем фото: если ExistingPhotos != nil, в БД оставляем existingPhotos ∪ newPhotoURLs;
// если ExistingPhotos == nil, просто добавляем newPhotoURLs (старые не трогаем).
// На выходе возвращаем актуальный список image_url из property_images после успешного сохранения.
func (s *PropertyService) UpdateOwned(ctx context.Context, userID, propertyID int, payload models.UpdatePropertyPayload, newPhotoURLs []string) ([]string, error) {
	if !payload.HasMetaChanges() && len(newPhotoURLs) == 0 {
		return nil, ErrEmptyPropertyPatch
	}
	ownerID, base, err := s.repo.LoadPropertyForEdit(ctx, propertyID)
	if err != nil {
		return nil, err
	}
	if ownerID != userID {
		return nil, repository.ErrPropertyForbidden
	}
	models.ApplyPropertyPatch(&base, payload.UpdatePropertyPatch)
	if !isPropertyTypeAllowed(base.Category, base.PropertyType) {
		return nil, ErrInvalidCategoryPropertyType
	}

	var toDelete []string
	if payload.ExistingPhotos != nil {
		dbURLs, err := s.repo.ListPropertyImageURLs(ctx, propertyID)
		if err != nil {
			return nil, err
		}
		keep := *payload.ExistingPhotos
		for _, dbu := range dbURLs {
			if !urlMatchesExistingPhotos(dbu, keep) {
				toDelete = append(toDelete, dbu)
			}
		}
		log.Printf("[properties] PATCH photos property_id=%d: dbPhotos=%v photosToDelete=%v newFilesCount=%d",
			propertyID, dbURLs, toDelete, len(newPhotoURLs))
	} else {
		log.Printf("[properties] PATCH photos property_id=%d: existingPhotos omitted (append-only); newFilesCount=%d", propertyID, len(newPhotoURLs))
	}

	err = s.repo.UpdatePropertyOwnedWithPhotos(ctx, propertyID, userID, base, toDelete, newPhotoURLs)
	if err != nil {
		return nil, err
	}
	for _, u := range toDelete {
		if p := localUploadPath(u); p != "" {
			_ = os.Remove(p)
		}
	}
	final, err := s.repo.ListPropertyImageURLs(ctx, propertyID)
	if err != nil {
		return nil, err
	}
	log.Printf("[properties] PATCH photos property_id=%d: finalPhotos=%v", propertyID, final)
	return final, nil
}

// Возвращает true, если dbURL есть в existingPhotos (фронт может прислать полный путь или только имя файла).
func urlMatchesExistingPhotos(dbURL string, existingPhotos []string) bool {
	dbURL = strings.TrimSpace(dbURL)
	for _, k := range existingPhotos {
		if urlMatchesSingle(dbURL, strings.TrimSpace(k)) {
			return true
		}
	}
	return false
}

func normalizePhotoRef(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if i := strings.IndexByte(s, '?'); i >= 0 {
		s = s[:i]
	}
	if strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") {
		if u, err := url.Parse(s); err == nil && u.Path != "" {
			return u.Path
		}
	}
	return s
}

func urlMatchesSingle(dbURL, k string) bool {
	if k == "" {
		return false
	}
	dbURL = strings.TrimSpace(dbURL)
	k = strings.TrimSpace(k)
	if dbURL == k {
		return true
	}
	nd := normalizePhotoRef(dbURL)
	nk := normalizePhotoRef(k)
	if nd != "" && nk != "" && nd == nk {
		return true
	}
	// Фронт иногда шлет только имя файла, а в БД лежит полный путь /uploads/properties/....
	if filepath.Base(nd) != "" && filepath.Base(nd) == filepath.Base(nk) {
		return true
	}
	if filepath.Base(dbURL) == k {
		return true
	}
	if strings.HasSuffix(dbURL, k) && (strings.HasPrefix(k, "/") || strings.Contains(k, "/")) {
		return true
	}
	if strings.HasSuffix(dbURL, "/"+k) {
		return true
	}
	return false
}

func localUploadPath(url string) string {
	if !strings.HasPrefix(url, "/uploads/") {
		return ""
	}
	return strings.TrimPrefix(url, "/")
}

func isPropertyTypeAllowed(category, propertyType string) bool {
	// Категорию ждем как "жилая" / "коммерческая" (если надо, потом добавим и residential/commercial).
	residential := map[string]bool{
		"квартира": true, "комната": true, "дом/дача": true, "коттедж": true,
	}
	commercial := map[string]bool{
		"офис": true, "коворкинг": true, "здание": true, "склад": true,
	}
	switch category {
	case "жилая":
		return residential[propertyType]
	case "коммерческая":
		return commercial[propertyType]
	default:
		// Если category не распознали, дальше обработчик вернет bad request.
		return false
	}
}

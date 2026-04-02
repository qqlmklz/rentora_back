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

// PropertyService handles property listing and CRUD.
type PropertyService struct {
	repo *repository.DB
}

// NewPropertyService creates a PropertyService.
func NewPropertyService(repo *repository.DB) *PropertyService {
	return &PropertyService{repo: repo}
}

var (
	ErrInvalidCategoryPropertyType = errors.New("invalid category/propertyType combination")
	ErrEmptyPropertyPatch          = errors.New("empty property patch")
)

// CatalogFilters describes incoming filters for catalog.
type CatalogFilters struct {
	Category     string
	PropertyType string
	Rooms        int
	PriceFrom    int
	PriceTo      int
	Location     string
	Sort         string
}

// ListForCatalog returns properties for /catalog with filters and sort applied.
func (s *PropertyService) ListForCatalog(ctx context.Context, f CatalogFilters) ([]models.Property, error) {
	return s.repo.ListProperties(ctx, repository.PropertyFilters{
		Category:     f.Category,
		PropertyType: f.PropertyType,
		Rooms:        f.Rooms,
		PriceFrom:    f.PriceFrom,
		PriceTo:      f.PriceTo,
		Location:     f.Location,
		Sort:         f.Sort,
	})
}

// Create creates property and saves image urls. Validates category and propertyType.
func (s *PropertyService) Create(ctx context.Context, userID int, in models.CreatePropertyInput, imageURLs []string) (int, error) {
	if !isPropertyTypeAllowed(in.Category, in.PropertyType) {
		return 0, ErrInvalidCategoryPropertyType
	}
	return s.repo.CreatePropertyWithImages(ctx, userID, in, imageURLs)
}

// GetByID returns a single property for the public detail page.
func (s *PropertyService) GetByID(ctx context.Context, id int) (*models.PropertyDetail, error) {
	return s.repo.GetPropertyByID(ctx, id)
}

// ListMine returns properties owned by the user (card shape).
func (s *PropertyService) ListMine(ctx context.Context, userID int) ([]models.Property, error) {
	return s.repo.ListPropertiesByUserID(ctx, userID)
}

// DeleteOwned deletes a property if it belongs to userID.
func (s *PropertyService) DeleteOwned(ctx context.Context, userID, propertyID int) error {
	return s.repo.DeletePropertyOwned(ctx, propertyID, userID)
}

// UpdateOwned applies payload and syncs photos: if ExistingPhotos != nil, DB becomes existingPhotos ∪ newPhotoURLs;
// if ExistingPhotos == nil, only appends newPhotoURLs (старые не трогаем).
// Returns актуальный список image_url из property_images после успешного сохранения.
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

// urlMatchesExistingPhotos returns true if dbURL is listed in existingPhotos (фронт может прислать полный путь или только имя файла).
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
	// фронт может прислать только имя файла, в БД — путь /uploads/properties/...
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
	// category: "жилая" / "коммерческая" (accept also "residential"/"commercial" later if needed)
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
		// if category not recognized, let handler treat it as bad request
		return false
	}
}


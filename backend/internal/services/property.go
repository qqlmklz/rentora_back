package services

import (
	"context"
	"errors"

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


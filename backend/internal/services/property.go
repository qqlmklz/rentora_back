package services

import (
	"context"

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


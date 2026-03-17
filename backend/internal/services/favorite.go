package services

import (
	"context"

	"rentora/backend/internal/models"
	"rentora/backend/internal/repository"
)

// FavoritesService handles user favorites.
type FavoritesService struct {
	repo *repository.DB
}

// NewFavoritesService creates FavoritesService.
func NewFavoritesService(repo *repository.DB) *FavoritesService {
	return &FavoritesService{repo: repo}
}

// Aliases to repository errors for handlers.
var (
	ErrFavoriteExists   = repository.ErrFavoriteExists
	ErrPropertyNotFound = repository.ErrPropertyNotFound
)

// List returns user's favorite properties.
func (s *FavoritesService) List(ctx context.Context, userID int) ([]models.Property, error) {
	return s.repo.ListFavorites(ctx, userID)
}

// Add adds property to user's favorites.
func (s *FavoritesService) Add(ctx context.Context, userID, propertyID int) error {
	return s.repo.AddFavorite(ctx, userID, propertyID)
}

// Remove removes property from user's favorites.
func (s *FavoritesService) Remove(ctx context.Context, userID, propertyID int) error {
	return s.repo.RemoveFavorite(ctx, userID, propertyID)
}


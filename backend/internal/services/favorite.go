package services

import (
	"context"

	"rentora/backend/internal/models"
	"rentora/backend/internal/repository"
)

// Сервис для работы с избранным пользователя.
type FavoritesService struct {
	repo *repository.DB
}

// Конструктор FavoritesService.
func NewFavoritesService(repo *repository.DB) *FavoritesService {
	return &FavoritesService{repo: repo}
}

// Просто алиасы ошибок из repository, чтобы удобно было в handlers.
var (
	ErrFavoriteExists    = repository.ErrFavoriteExists
	ErrPropertyNotFound  = repository.ErrPropertyNotFound
	ErrPropertyForbidden = repository.ErrPropertyForbidden
)

// Возвращаем список избранных объявлений пользователя.
func (s *FavoritesService) List(ctx context.Context, userID int) ([]models.Property, error) {
	return s.repo.ListFavorites(ctx, userID)
}

// Добавляем объявление в избранное пользователя.
func (s *FavoritesService) Add(ctx context.Context, userID, propertyID int) error {
	return s.repo.AddFavorite(ctx, userID, propertyID)
}

// Удаляем объявление из избранного пользователя.
func (s *FavoritesService) Remove(ctx context.Context, userID, propertyID int) error {
	return s.repo.RemoveFavorite(ctx, userID, propertyID)
}


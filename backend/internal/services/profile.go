package services

import (
	"context"
	"errors"

	"rentora/backend/internal/models"
	"rentora/backend/internal/repository"

	"golang.org/x/crypto/bcrypt"
)

// Сервис операций с профилем пользователя.
type ProfileService struct {
	repo *repository.DB
}

// Конструктор ProfileService.
func NewProfileService(repo *repository.DB) *ProfileService {
	return &ProfileService{repo: repo}
}

// Ошибка, когда при обновлении профиля email уже занят другим пользователем.
var ErrEmailTaken = errors.New("email already taken")

// Ошибка, когда текущий пароль не совпал.
var ErrWrongPassword = errors.New("wrong current password")

// Возвращаем пользователя по ID (вызывающий код сам решает, что делать если его нет).
func (s *ProfileService) GetProfile(ctx context.Context, userID int) (*models.User, error) {
	return s.repo.GetUserByID(ctx, userID)
}

// Обновляем name/email/phone. Если email занят другим пользователем, вернем ErrEmailTaken.
func (s *ProfileService) UpdateProfile(ctx context.Context, userID int, name, email string, phone *string) error {
	other, err := s.repo.GetUserByEmailExcludingID(ctx, email, userID)
	if err != nil {
		return err
	}
	if other != nil {
		return ErrEmailTaken
	}
	return s.repo.UpdateProfile(ctx, userID, name, email, phone)
}

// Сохраняем путь к аватару пользователя.
func (s *ProfileService) UpdateAvatar(ctx context.Context, userID int, avatarPath string) error {
	return s.repo.UpdateAvatar(ctx, userID, &avatarPath)
}

// Очищаем аватар пользователя.
func (s *ProfileService) DeleteAvatar(ctx context.Context, userID int) error {
	return s.repo.UpdateAvatar(ctx, userID, nil)
}

// Проверяем текущий пароль и ставим новый. Если текущий неверный, вернем ErrWrongPassword.
func (s *ProfileService) UpdatePassword(ctx context.Context, userID int, currentPassword, newPassword string) error {
	u, err := s.repo.GetUserByID(ctx, userID)
	if err != nil || u == nil {
		return err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(currentPassword)); err != nil {
		return ErrWrongPassword
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), 12)
	if err != nil {
		return err
	}
	return s.repo.UpdatePassword(ctx, userID, string(hash))
}

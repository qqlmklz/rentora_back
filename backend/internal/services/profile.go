package services

import (
	"context"
	"errors"

	"rentora/backend/internal/models"
	"rentora/backend/internal/repository"

	"golang.org/x/crypto/bcrypt"
)

// ProfileService handles profile operations.
type ProfileService struct {
	repo *repository.DB
}

// NewProfileService creates a ProfileService.
func NewProfileService(repo *repository.DB) *ProfileService {
	return &ProfileService{repo: repo}
}

// ErrEmailTaken is returned when updating profile with an email that belongs to another user.
var ErrEmailTaken = errors.New("email already taken")

// ErrWrongPassword is returned when current password does not match.
var ErrWrongPassword = errors.New("wrong current password")

// GetProfile returns user by ID (caller must ensure user exists).
func (s *ProfileService) GetProfile(ctx context.Context, userID int) (*models.User, error) {
	return s.repo.GetUserByID(ctx, userID)
}

// UpdateProfile updates name, email, phone. Returns ErrEmailTaken if email is taken by another user.
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

// UpdateAvatar saves avatar path for user.
func (s *ProfileService) UpdateAvatar(ctx context.Context, userID int, avatarPath string) error {
	return s.repo.UpdateAvatar(ctx, userID, &avatarPath)
}

// DeleteAvatar clears avatar for user.
func (s *ProfileService) DeleteAvatar(ctx context.Context, userID int) error {
	return s.repo.UpdateAvatar(ctx, userID, nil)
}

// UpdatePassword verifies current password and sets new one. Returns ErrWrongPassword if current is wrong.
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

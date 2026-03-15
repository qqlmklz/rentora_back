package services

import (
	"context"
	"errors"

	"rentora/backend/internal/models"
	"rentora/backend/internal/repository"
	"rentora/backend/internal/utils"

	"golang.org/x/crypto/bcrypt"
)

// AuthService handles login, register, token refresh.
type AuthService struct {
	repo   *repository.DB
	secret string
}

// NewAuthService creates an AuthService.
func NewAuthService(repo *repository.DB, jwtSecret string) *AuthService {
	return &AuthService{repo: repo, secret: jwtSecret}
}

// ErrUserExists is returned when registering with an existing email.
var ErrUserExists = errors.New("user with this email already exists")

// ErrInvalidCredentials is returned when login email or password is wrong.
var ErrInvalidCredentials = errors.New("invalid email or password")

const bcryptCost = 12

// Register creates a new user. Returns ErrUserExists if email is taken.
func (s *AuthService) Register(ctx context.Context, name, email, password string) error {
	existing, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		return err
	}
	if existing != nil {
		return ErrUserExists
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return err
	}

	u := &models.User{
		Name:         name,
		Email:        email,
		PasswordHash: string(hash),
	}
	err = s.repo.CreateUser(ctx, u)
	if err != nil {
		if err == repository.ErrDuplicateEmail {
			return ErrUserExists
		}
		return err
	}
	return nil
}

// Login finds user by email, checks password, returns user and JWT or ErrInvalidCredentials.
func (s *AuthService) Login(ctx context.Context, email, password string) (*models.User, string, error) {
	u, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, "", err
	}
	if u == nil {
		return nil, "", ErrInvalidCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return nil, "", ErrInvalidCredentials
	}
	token, err := utils.NewToken(u.ID, s.secret)
	if err != nil {
		return nil, "", err
	}
	return u, token, nil
}

// GetUserByID returns user by ID (for /me). Returns nil if not found.
func (s *AuthService) GetUserByID(ctx context.Context, id int) (*models.User, error) {
	return s.repo.GetUserByID(ctx, id)
}

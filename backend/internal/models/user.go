package models

import "time"

// Модель строки пользователя в базе.
type User struct {
	ID           int       `json:"id"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	Phone        *string   `json:"phone,omitempty"`
	PasswordHash string    `json:"-"`
	Avatar       *string   `json:"avatar,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Формируем UserResponse для API-ответов (без пароля).
func (u *User) ToResponse() UserResponse {
	return UserResponse{ID: u.ID, Name: u.Name, Email: u.Email, Phone: u.Phone, Avatar: u.Avatar}
}

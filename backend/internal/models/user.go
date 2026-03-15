package models

import "time"

// User represents a user row in the database.
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

// ToResponse returns a UserResponse for API responses (no password).
func (u *User) ToResponse() UserResponse {
	return UserResponse{ID: u.ID, Name: u.Name, Email: u.Email, Phone: u.Phone, Avatar: u.Avatar}
}

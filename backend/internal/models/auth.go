package models

// RegisterRequest is the body for POST /api/auth/register.
type RegisterRequest struct {
	Name     string `json:"name" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

// LoginRequest is the body for POST /api/auth/login.
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse is the successful response for POST /api/auth/login.
type LoginResponse struct {
	Token string      `json:"token"`
	User  UserResponse `json:"user"`
}

// UserResponse is the user object in API responses (no password, no internal fields).
type UserResponse struct {
	ID     int     `json:"id"`
	Name   string  `json:"name"`
	Email  string  `json:"email"`
	Phone  *string `json:"phone"`
	Avatar *string `json:"avatar"`
}

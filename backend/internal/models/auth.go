package models

// Тело запроса для POST /api/auth/register.
type RegisterRequest struct {
	Name     string `json:"name" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

// Тело запроса для POST /api/auth/login.
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// Успешный ответ для POST /api/auth/login.
type LoginResponse struct {
	Token string      `json:"token"`
	User  UserResponse `json:"user"`
}

// Объект пользователя в API-ответах (без пароля и внутренних полей).
type UserResponse struct {
	ID     int     `json:"id"`
	Name   string  `json:"name"`
	Email  string  `json:"email"`
	Phone  *string `json:"phone"`
	Avatar *string `json:"avatar"`
}

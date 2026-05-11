package models

// Тело запроса для PATCH /api/profile.
type UpdateProfileRequest struct {
	Name  *string `json:"name"`
	Email *string `json:"email"`
	Phone *string `json:"phone"`
}

// Тело запроса для PATCH /api/profile/password.
type UpdatePasswordRequest struct {
	CurrentPassword string `json:"currentPassword" binding:"required"`
	NewPassword     string `json:"newPassword" binding:"required,min=6"`
}

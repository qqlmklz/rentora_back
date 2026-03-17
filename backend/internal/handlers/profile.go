package handlers

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"rentora/backend/internal/middleware"
	"rentora/backend/internal/models"
	"rentora/backend/internal/services"
	"rentora/backend/internal/utils"

	"github.com/gin-gonic/gin"
)

const (
	avatarFormKey   = "avatar"
	avatarMaxSize   = 5 << 20 // 5MB
	avatarUploadDir = "uploads/avatars"
	avatarURLPrefix = "/uploads/"
)

var allowedAvatarTypes = map[string]bool{
	"image/jpeg": true, "image/png": true, "image/gif": true, "image/webp": true,
}

// profileResponse returns UserResponse with avatar as full URL path when set.
func profileResponse(u *models.User) models.UserResponse {
	r := u.ToResponse()
	if u.Avatar != nil && *u.Avatar != "" {
		// Backward compatible:
		// - new format stored in DB: "/uploads/avatars/filename.jpg"
		// - old format stored in DB: "avatars/filename.jpg"
		if strings.HasPrefix(*u.Avatar, "/") {
			r.Avatar = u.Avatar
		} else {
			url := avatarURLPrefix + *u.Avatar
			r.Avatar = &url
		}
	}
	return r
}

// GetProfile returns the current user's profile (id, name, email, phone, avatar).
func GetProfile(profileService *services.ProfileService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := middleware.GetUserID(c)
		if !ok {
			utils.JSONErrorUnauthorized(c, "Требуется авторизация")
			return
		}
		user, err := profileService.GetProfile(c.Request.Context(), userID)
		if err != nil {
			log.Printf("[profile] get error: %v", err)
			utils.JSONErrorInternal(c, "Ошибка получения профиля")
			return
		}
		if user == nil {
			utils.JSONErrorUnauthorized(c, "Пользователь не найден")
			return
		}
		c.JSON(http.StatusOK, profileResponse(user))
	}
}

// UpdateProfile updates name, email, phone.
func UpdateProfile(profileService *services.ProfileService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := middleware.GetUserID(c)
		if !ok {
			utils.JSONErrorUnauthorized(c, "Требуется авторизация")
			return
		}
		user, err := profileService.GetProfile(c.Request.Context(), userID)
		if err != nil || user == nil {
			utils.JSONErrorUnauthorized(c, "Пользователь не найден")
			return
		}
		var req models.UpdateProfileRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			utils.JSONErrorBadRequest(c, "Неверный формат данных")
			return
		}
		name, email := user.Name, user.Email
		phone := user.Phone
		if req.Name != nil {
			name = *req.Name
		}
		if req.Email != nil {
			email = *req.Email
		}
		if req.Phone != nil {
			phone = req.Phone
		}
		if name == "" || email == "" {
			utils.JSONErrorBadRequest(c, "name и email обязательны")
			return
		}
		err = profileService.UpdateProfile(c.Request.Context(), userID, name, email, phone)
		if err != nil {
			if err == services.ErrEmailTaken {
				c.AbortWithStatusJSON(http.StatusConflict, gin.H{"message": "Пользователь с таким email уже существует"})
				return
			}
			log.Printf("[profile] update error: %v", err)
			utils.JSONErrorInternal(c, "Ошибка обновления профиля")
			return
		}
		user.Name, user.Email, user.Phone = name, email, phone
		c.JSON(http.StatusOK, profileResponse(user))
	}
}

// UpdateAvatar handles multipart file upload and saves avatar path.
func UpdateAvatar(profileService *services.ProfileService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := middleware.GetUserID(c)
		if !ok {
			utils.JSONErrorUnauthorized(c, "Требуется авторизация")
			return
		}
		file, err := c.FormFile(avatarFormKey)
		if err != nil {
			utils.JSONErrorBadRequest(c, "Файл аватарки не найден (ожидается поле avatar)")
			return
		}
		if file.Size > avatarMaxSize {
			utils.JSONErrorBadRequest(c, "Размер файла не должен превышать 5 МБ")
			return
		}
		opened, err := file.Open()
		if err != nil {
			log.Printf("[profile] avatar open: %v", err)
			utils.JSONErrorInternal(c, "Ошибка загрузки файла")
			return
		}
		buf := make([]byte, 512)
		n, _ := opened.Read(buf)
		opened.Close()
		contentType := http.DetectContentType(buf[:n])
		if !allowedAvatarTypes[contentType] {
			utils.JSONErrorBadRequest(c, "Допустимые форматы: JPEG, PNG, GIF, WebP")
			return
		}
		ext := ".jpg"
		switch contentType {
		case "image/png":
			ext = ".png"
		case "image/gif":
			ext = ".gif"
		case "image/webp":
			ext = ".webp"
		}
		baseName := fmt.Sprintf("%d_%d%s", userID, time.Now().UnixNano(), ext)
		avatarPath := filepath.Join(avatarUploadDir, baseName)
		publicPath := "/uploads/avatars/" + baseName
		if err := os.MkdirAll(avatarUploadDir, 0755); err != nil {
			log.Printf("[profile] mkdir: %v", err)
			utils.JSONErrorInternal(c, "Ошибка сохранения файла")
			return
		}
		if err := c.SaveUploadedFile(file, avatarPath); err != nil {
			log.Printf("[profile] avatar save: %v", err)
			utils.JSONErrorInternal(c, "Ошибка сохранения файла")
			return
		}
		if err := profileService.UpdateAvatar(c.Request.Context(), userID, publicPath); err != nil {
			log.Printf("[profile] avatar update: %v", err)
			utils.JSONErrorInternal(c, "Ошибка обновления аватарки")
			return
		}
		user, err := profileService.GetProfile(c.Request.Context(), userID)
		if err != nil || user == nil {
			utils.JSONErrorInternal(c, "Ошибка получения профиля")
			return
		}
		c.JSON(http.StatusOK, profileResponse(user))
	}
}

// DeleteAvatar removes avatar for the current user.
func DeleteAvatar(profileService *services.ProfileService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := middleware.GetUserID(c)
		if !ok {
			utils.JSONErrorUnauthorized(c, "Требуется авторизация")
			return
		}
		if err := profileService.DeleteAvatar(c.Request.Context(), userID); err != nil {
			log.Printf("[profile] delete avatar: %v", err)
			utils.JSONErrorInternal(c, "Ошибка удаления аватарки")
			return
		}
		user, err := profileService.GetProfile(c.Request.Context(), userID)
		if err != nil || user == nil {
			utils.JSONErrorInternal(c, "Ошибка получения профиля")
			return
		}
		c.JSON(http.StatusOK, profileResponse(user))
	}
}

// UpdatePassword changes password after verifying current one.
func UpdatePassword(profileService *services.ProfileService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := middleware.GetUserID(c)
		if !ok {
			utils.JSONErrorUnauthorized(c, "Требуется авторизация")
			return
		}
		var req models.UpdatePasswordRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			utils.JSONErrorBadRequest(c, "Укажите currentPassword и newPassword (не менее 6 символов)")
			return
		}
		err := profileService.UpdatePassword(c.Request.Context(), userID, req.CurrentPassword, req.NewPassword)
		if err != nil {
			if err == services.ErrWrongPassword {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "Неверный текущий пароль"})
				return
			}
			log.Printf("[profile] password: %v", err)
			utils.JSONErrorInternal(c, "Ошибка смены пароля")
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Пароль успешно изменён"})
	}
}

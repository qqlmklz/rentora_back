package handlers

import (
	"log"
	"net/http"

	"rentora/backend/internal/middleware"
	"rentora/backend/internal/models"
	"rentora/backend/internal/services"
	"rentora/backend/internal/utils"

	"github.com/gin-gonic/gin"
)

// Register creates a user. Expects AuthService to be provided via closure.
func Register(authService *services.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req models.RegisterRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			log.Printf("[register] invalid body: %v", err)
			utils.JSONErrorBadRequest(c, "Неверный формат запроса: укажите name, email и password (пароль не менее 6 символов)")
			return
		}

		err := authService.Register(c.Request.Context(), req.Name, req.Email, req.Password)
		if err != nil {
			if err == services.ErrUserExists {
				c.AbortWithStatusJSON(http.StatusConflict, gin.H{
					"message": "Пользователь с таким email уже существует",
				})
				return
			}
			log.Printf("[register] error: %v", err)
			utils.JSONErrorInternal(c, "Не удалось зарегистрировать пользователя")
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"message": "Пользователь успешно зарегистрирован",
		})
	}
}

// Login authenticates by email/password and returns JWT and user.
func Login(authService *services.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req models.LoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			log.Printf("[login] invalid body: %v", err)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "Неверный email или пароль"})
			return
		}

		user, token, err := authService.Login(c.Request.Context(), req.Email, req.Password)
		if err != nil {
			if err == services.ErrInvalidCredentials {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "Неверный email или пароль"})
				return
			}
			log.Printf("[login] error: %v", err)
			utils.JSONErrorInternal(c, "Ошибка авторизации")
			return
		}

		c.JSON(http.StatusOK, models.LoginResponse{
			Token: token,
			User:  user.ToResponse(),
		})
	}
}

// Me returns the current user from JWT.
func Me(authService *services.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := middleware.GetUserID(c)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "Неверный email или пароль"})
			return
		}
		user, err := authService.GetUserByID(c.Request.Context(), userID)
		if err != nil {
			log.Printf("[me] error: %v", err)
			utils.JSONErrorInternal(c, "Ошибка получения пользователя")
			return
		}
		if user == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "Пользователь не найден"})
			return
		}
		c.JSON(http.StatusOK, user.ToResponse())
	}
}

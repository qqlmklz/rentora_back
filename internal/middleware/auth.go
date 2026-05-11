package middleware

import (
	"net/http"
	"strings"

	"rentora/backend/internal/utils"

	"github.com/gin-gonic/gin"
)

const userIDKey = "user_id"

// Проверяем JWT и кладем user_id в контекст. Если токен отсутствует или битый — 401.
func Auth(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "Неверный email или пароль"})
			return
		}
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "Неверный email или пароль"})
			return
		}
		userID, err := utils.ParseToken(parts[1], jwtSecret)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "Неверный email или пароль"})
			return
		}
		c.Set(userIDKey, userID)
		c.Next()
	}
}

// Достаем user ID, который положил Auth middleware. Использовать после Auth.
func GetUserID(c *gin.Context) (int, bool) {
	id, ok := c.Get(userIDKey)
	if !ok {
		return 0, false
	}
	uid, ok := id.(int)
	return uid, ok
}

// Пытаемся распарсить Authorization: Bearer <jwt>, не заваливая весь запрос.
// Если заголовка нет или токен невалидный, вернем (0, false).
func ParseUserIDFromBearer(c *gin.Context, jwtSecret string) (int, bool) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return 0, false
	}
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return 0, false
	}
	userID, err := utils.ParseToken(parts[1], jwtSecret)
	if err != nil {
		return 0, false
	}
	return userID, true
}

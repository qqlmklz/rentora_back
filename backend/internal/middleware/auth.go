package middleware

import (
	"net/http"
	"strings"

	"rentora/backend/internal/utils"

	"github.com/gin-gonic/gin"
)

const userIDKey = "user_id"

// Auth validates JWT and sets user_id in context. Returns 401 if missing or invalid.
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

// GetUserID returns the user ID set by Auth middleware. Must be used after Auth.
func GetUserID(c *gin.Context) (int, bool) {
	id, ok := c.Get(userIDKey)
	if !ok {
		return 0, false
	}
	uid, ok := id.(int)
	return uid, ok
}

// ParseUserIDFromBearer parses Authorization: Bearer <jwt> without failing the request.
// Returns (0, false) if missing or invalid.
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

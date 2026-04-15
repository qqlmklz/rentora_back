package middleware

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Мидлварка, которая ловит panic и отдает JSON-ошибку.
func RecoveryJSON() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("panic recovered: %v", err)
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error":   "Internal Server Error",
					"message": "An unexpected error occurred",
				})
			}
		}()
		c.Next()
	}
}

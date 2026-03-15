package middleware

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// RecoveryJSON returns a middleware that recovers from panics and responds with JSON.
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

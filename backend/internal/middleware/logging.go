package middleware

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

// Logging returns a middleware that logs HTTP method, path, client IP, latency and status.
func Logging() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		clientIP := c.ClientIP()
		method := c.Request.Method

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()
		log.Printf("[%s] %s %s %d %v", method, path, clientIP, status, latency)
	}
}

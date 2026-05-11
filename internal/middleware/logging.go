package middleware

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

// Мидлварка для логов: метод, путь, IP клиента, задержка и статус.
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

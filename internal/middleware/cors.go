package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// Мидлварка для CORS: выставляет заголовки по списку разрешенных origin.
func CORS(origins []string) gin.HandlerFunc {
	allowAll := false
	for _, o := range origins {
		if strings.TrimSpace(o) == "*" {
			allowAll = true
			break
		}
	}

	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		if origin == "" {
			origin = "*"
		}

		if allowAll {
			c.Header("Access-Control-Allow-Origin", "*")
		} else {
			for _, o := range origins {
				o = strings.TrimSpace(o)
				if o == "*" || strings.EqualFold(o, origin) {
					if o != "*" {
						c.Header("Access-Control-Allow-Origin", origin)
					} else {
						c.Header("Access-Control-Allow-Origin", "*")
					}
					break
				}
			}
		}

		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

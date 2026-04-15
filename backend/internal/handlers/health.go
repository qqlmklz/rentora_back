package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Тело ответа для GET /api/health.
type HealthResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// Хендлер для health-check эндпоинта.
func Health(c *gin.Context) {
	c.JSON(http.StatusOK, HealthResponse{
		Status:  "ok",
		Message: "Rentora backend is running",
	})
}

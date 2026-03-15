package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// HealthResponse is the response body for GET /api/health.
type HealthResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// Health returns a handler for the health check endpoint.
func Health(c *gin.Context) {
	c.JSON(http.StatusOK, HealthResponse{
		Status:  "ok",
		Message: "Rentora backend is running",
	})
}

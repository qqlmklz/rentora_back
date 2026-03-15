package utils

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ErrorResponse is a standard JSON error body.
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

// JSONError sends a JSON error response and aborts.
func JSONError(c *gin.Context, status int, errMsg, message string) {
	c.AbortWithStatusJSON(status, ErrorResponse{Error: errMsg, Message: message})
}

// JSONErrorBadRequest sends 400 with message.
func JSONErrorBadRequest(c *gin.Context, message string) {
	JSONError(c, http.StatusBadRequest, "Bad Request", message)
}

// JSONErrorNotFound sends 404 with message.
func JSONErrorNotFound(c *gin.Context, message string) {
	JSONError(c, http.StatusNotFound, "Not Found", message)
}

// JSONErrorUnauthorized sends 401 with message.
func JSONErrorUnauthorized(c *gin.Context, message string) {
	JSONError(c, http.StatusUnauthorized, "Unauthorized", message)
}

// JSONErrorInternal sends 500 with message.
func JSONErrorInternal(c *gin.Context, message string) {
	JSONError(c, http.StatusInternalServerError, "Internal Server Error", message)
}

// JSONErrorConflict sends 409 with message.
func JSONErrorConflict(c *gin.Context, message string) {
	JSONError(c, http.StatusConflict, "Conflict", message)
}

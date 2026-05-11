package utils

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Стандартное тело JSON-ошибки.
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

// Отдаем JSON-ошибку и прерываем обработку запроса.
func JSONError(c *gin.Context, status int, errMsg, message string) {
	c.AbortWithStatusJSON(status, ErrorResponse{Error: errMsg, Message: message})
}

// Возвращаем 400 с сообщением.
func JSONErrorBadRequest(c *gin.Context, message string) {
	JSONError(c, http.StatusBadRequest, "Bad Request", message)
}

// Возвращаем 404 с сообщением.
func JSONErrorNotFound(c *gin.Context, message string) {
	JSONError(c, http.StatusNotFound, "Not Found", message)
}

// Возвращаем 403 с сообщением.
func JSONErrorForbidden(c *gin.Context, message string) {
	JSONError(c, http.StatusForbidden, "Forbidden", message)
}

// Возвращаем 401 с сообщением.
func JSONErrorUnauthorized(c *gin.Context, message string) {
	JSONError(c, http.StatusUnauthorized, "Unauthorized", message)
}

// Возвращаем 500 с сообщением.
func JSONErrorInternal(c *gin.Context, message string) {
	JSONError(c, http.StatusInternalServerError, "Internal Server Error", message)
}

// Возвращаем 409 с сообщением.
func JSONErrorConflict(c *gin.Context, message string) {
	JSONError(c, http.StatusConflict, "Conflict", message)
}

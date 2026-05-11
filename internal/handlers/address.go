package handlers

import (
	"net/http"

	"rentora/backend/internal/services"

	"github.com/gin-gonic/gin"
)

type addressSuggestionsRequest struct {
	Query string `json:"query"`
}

func GetAddressSuggestions(addressService *services.AddressService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req addressSuggestionsRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusOK, gin.H{"suggestions": []services.AddressSuggestion{}})
			return
		}

		suggestions := addressService.Suggest(c.Request.Context(), req.Query)
		c.JSON(http.StatusOK, gin.H{
			"suggestions": suggestions,
		})
	}
}

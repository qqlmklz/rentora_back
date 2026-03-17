package handlers

import (
	"net/http"
	"strconv"

	"rentora/backend/internal/services"

	"github.com/gin-gonic/gin"
)

// GetProperties handles GET /api/properties for catalog.
func GetProperties(propertyService *services.PropertyService) gin.HandlerFunc {
	return func(c *gin.Context) {
		filters := services.CatalogFilters{
			Category:     c.Query("category"),
			PropertyType: c.Query("propertyType"),
			Location:     c.Query("location"),
			Sort:         c.DefaultQuery("sort", "newest"),
		}

		if roomsStr := c.Query("rooms"); roomsStr != "" {
			if v, err := strconv.Atoi(roomsStr); err == nil {
				filters.Rooms = v
			}
		}
		if priceFromStr := c.Query("priceFrom"); priceFromStr != "" {
			if v, err := strconv.Atoi(priceFromStr); err == nil {
				filters.PriceFrom = v
			}
		}
		if priceToStr := c.Query("priceTo"); priceToStr != "" {
			if v, err := strconv.Atoi(priceToStr); err == nil {
				filters.PriceTo = v
			}
		}

		props, err := propertyService.ListForCatalog(c.Request.Context(), filters)
		if err != nil {
			// For catalog we just return generic 500; logging can be added later.
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "Ошибка загрузки объявлений"})
			return
		}
		c.JSON(http.StatusOK, props)
	}
}


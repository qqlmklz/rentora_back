package handlers

import (
	"log"
	"net/http"
	"strconv"

	"rentora/backend/internal/middleware"
	"rentora/backend/internal/services"
	"rentora/backend/internal/utils"

	"github.com/gin-gonic/gin"
)

// GetFavorites returns all favorite properties for current user.
func GetFavorites(favService *services.FavoritesService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := middleware.GetUserID(c)
		if !ok {
			utils.JSONErrorUnauthorized(c, "Требуется авторизация")
			return
		}
		props, err := favService.List(c.Request.Context(), userID)
		if err != nil {
			log.Printf("[favorites] list: %v", err)
			utils.JSONErrorInternal(c, "Ошибка получения избранного")
			return
		}
		c.JSON(http.StatusOK, props)
	}
}

// AddFavorite adds property to user's favorites.
func AddFavorite(favService *services.FavoritesService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := middleware.GetUserID(c)
		if !ok {
			utils.JSONErrorUnauthorized(c, "Требуется авторизация")
			return
		}
		propertyID, err := strconv.Atoi(c.Param("propertyId"))
		if err != nil || propertyID <= 0 {
			utils.JSONErrorBadRequest(c, "Некорректный идентификатор объявления")
			return
		}
		err = favService.Add(c.Request.Context(), userID, propertyID)
		if err != nil {
			switch err {
			case services.ErrPropertyNotFound:
				utils.JSONErrorNotFound(c, "Объявление не найдено")
			case services.ErrFavoriteExists:
				utils.JSONErrorConflict(c, "Объявление уже в избранном")
			default:
				log.Printf("[favorites] add: %v", err)
				utils.JSONErrorInternal(c, "Ошибка добавления в избранное")
			}
			return
		}
		c.Status(http.StatusNoContent)
	}
}

// RemoveFavorite removes property from user's favorites.
func RemoveFavorite(favService *services.FavoritesService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := middleware.GetUserID(c)
		if !ok {
			utils.JSONErrorUnauthorized(c, "Требуется авторизация")
			return
		}
		propertyID, err := strconv.Atoi(c.Param("propertyId"))
		if err != nil || propertyID <= 0 {
			utils.JSONErrorBadRequest(c, "Некорректный идентификатор объявления")
			return
		}
		err = favService.Remove(c.Request.Context(), userID, propertyID)
		if err != nil {
			if err == services.ErrPropertyNotFound {
				utils.JSONErrorNotFound(c, "Объявление не найдено")
				return
			}
			log.Printf("[favorites] remove: %v", err)
			utils.JSONErrorInternal(c, "Ошибка удаления из избранного")
			return
		}
		c.Status(http.StatusNoContent)
	}
}


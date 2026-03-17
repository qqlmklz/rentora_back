package routes

import (
	"rentora/backend/internal/handlers"
	"rentora/backend/internal/middleware"
	"rentora/backend/internal/services"

	"github.com/gin-gonic/gin"
)

// Setup configures all routes and middleware on the given engine.
func Setup(r *gin.Engine, corsOrigins []string, authService *services.AuthService, profileService *services.ProfileService, propertyService *services.PropertyService, favoritesService *services.FavoritesService, jwtSecret string) {
	r.Use(middleware.RecoveryJSON())
	r.Use(middleware.Logging())
	r.Use(middleware.CORS(corsOrigins))

	r.Static("/uploads", "uploads")

	api := r.Group("/api")
	{
		api.GET("/health", handlers.Health)

		auth := api.Group("/auth")
		authRoutes(auth, authService, jwtSecret)

		profile := api.Group("/profile")
		profileRoutes(profile, profileService, jwtSecret)

		users := api.Group("/users")
		userRoutes(users)

		properties := api.Group("/properties")
		propertyRoutes(properties, propertyService)

		applications := api.Group("/applications")
		applicationRoutes(applications)

		favorites := api.Group("/favorites")
		favoriteRoutes(favorites, favoritesService, jwtSecret)
	}
}

func authRoutes(g *gin.RouterGroup, authService *services.AuthService, jwtSecret string) {
	g.POST("/register", handlers.Register(authService))
	g.POST("/login", handlers.Login(authService))
	g.GET("/me", middleware.Auth(jwtSecret), handlers.Me(authService))
}

func profileRoutes(g *gin.RouterGroup, profileService *services.ProfileService, jwtSecret string) {
	g.Use(middleware.Auth(jwtSecret))
	g.GET("", handlers.GetProfile(profileService))
	g.PATCH("", handlers.UpdateProfile(profileService))
	g.PATCH("/avatar", handlers.UpdateAvatar(profileService))
	g.DELETE("/avatar", handlers.DeleteAvatar(profileService))
	g.PATCH("/password", handlers.UpdatePassword(profileService))
}

func userRoutes(g *gin.RouterGroup) {
	// GET/PUT /api/users/:id, etc.
}

func propertyRoutes(g *gin.RouterGroup, propertyService *services.PropertyService) {
	// Catalog: list properties with filters.
	g.GET("", handlers.GetProperties(propertyService))
}

func applicationRoutes(g *gin.RouterGroup) {
	// GET/POST /api/applications, etc.
}

func favoriteRoutes(g *gin.RouterGroup, favService *services.FavoritesService, jwtSecret string) {
	g.Use(middleware.Auth(jwtSecret))
	g.GET("", handlers.GetFavorites(favService))
	g.POST("/:propertyId", handlers.AddFavorite(favService))
	g.DELETE("/:propertyId", handlers.RemoveFavorite(favService))
}

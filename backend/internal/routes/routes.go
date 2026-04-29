package routes

import (
	"rentora/backend/internal/handlers"
	"rentora/backend/internal/middleware"
	"rentora/backend/internal/services"
	"rentora/backend/internal/ws"

	"github.com/gin-gonic/gin"
)

// Тут настраиваем все роуты и middleware на переданном движке.
func Setup(r *gin.Engine, corsOrigins []string, authService *services.AuthService, profileService *services.ProfileService, propertyService *services.PropertyService, applicationService *services.ApplicationService, favoritesService *services.FavoritesService, chatService *services.ChatService, contractService *services.ContractService, hub *ws.Hub, jwtSecret string) {
	r.Use(middleware.RecoveryJSON())
	r.Use(middleware.Logging())
	r.Use(middleware.CORS(corsOrigins))

	r.Static("/uploads", "uploads")
	r.GET("/ws/chats", handlers.ChatWebSocket(hub, jwtSecret, corsOrigins))

	api := r.Group("/api")
	{
		api.GET("/health", handlers.Health)

		auth := api.Group("/auth")
		authRoutes(auth, authService, jwtSecret)

		profile := api.Group("/profile")
		profileRoutes(profile, profileService, propertyService, contractService, applicationService, jwtSecret)

		users := api.Group("/users")
		userRoutes(users)

		properties := api.Group("/properties")
		propertyRoutes(properties, propertyService, jwtSecret)
		api.GET("/listings/recommendations", middleware.Auth(jwtSecret), handlers.GetRecommendations(propertyService))

		requests := api.Group("/requests")
		requestRoutes(requests, applicationService, jwtSecret)

		favorites := api.Group("/favorites")
		favoriteRoutes(favorites, favoritesService, jwtSecret)

		chats := api.Group("/chats")
		chatRoutes(chats, chatService, contractService, jwtSecret)

		contracts := api.Group("/contracts")
		contracts.Use(middleware.Auth(jwtSecret))
		contracts.GET("/:id", handlers.GetContract(contractService))
		contracts.PATCH("/:id/accept", handlers.AcceptContract(contractService))
		contracts.PATCH("/:id/reject", handlers.RejectContract(contractService))
		contracts.PATCH("/:id/terminate", handlers.TerminateContract(contractService))
	}
}

func authRoutes(g *gin.RouterGroup, authService *services.AuthService, jwtSecret string) {
	g.POST("/register", handlers.Register(authService))
	g.POST("/login", handlers.Login(authService))
	g.GET("/me", middleware.Auth(jwtSecret), handlers.Me(authService))
}

func profileRoutes(g *gin.RouterGroup, profileService *services.ProfileService, propertyService *services.PropertyService, contractService *services.ContractService, applicationService *services.ApplicationService, jwtSecret string) {
	g.Use(middleware.Auth(jwtSecret))
	g.GET("", handlers.GetProfile(profileService))
	g.PATCH("", handlers.UpdateProfile(profileService))
	g.PATCH("/avatar", handlers.UpdateAvatar(profileService))
	g.DELETE("/avatar", handlers.DeleteAvatar(profileService))
	g.PATCH("/password", handlers.UpdatePassword(profileService))
	g.GET("/properties", handlers.GetMyProperties(propertyService))
	g.GET("/documents", handlers.GetProfileDocuments(contractService))
	g.GET("/requests", handlers.GetProfileRequests(applicationService))
}

func userRoutes(g *gin.RouterGroup) {
	// Здесь будут маршруты пользователей (GET/PUT /api/users/:id и другие).
}

func propertyRoutes(g *gin.RouterGroup, propertyService *services.PropertyService, jwtSecret string) {
	// Каталог: список объявлений с фильтрами.
	g.GET("", handlers.GetProperties(propertyService, jwtSecret))
	// Создание объявления: только для авторизованных.
	g.POST("", middleware.Auth(jwtSecret), handlers.CreateProperty(propertyService))
	// Одно объявление: публично, но с JWT можем показать поля только для владельца.
	g.GET("/:id", handlers.GetPropertyByID(propertyService, jwtSecret))
	// Действия только для владельца (JWT обязателен).
	g.DELETE("/:id", middleware.Auth(jwtSecret), handlers.DeleteProperty(propertyService))
	g.PATCH("/:id", middleware.Auth(jwtSecret), handlers.UpdateProperty(propertyService))
}

func requestRoutes(g *gin.RouterGroup, applicationService *services.ApplicationService, jwtSecret string) {
	g.Use(middleware.Auth(jwtSecret))
	g.GET("/available-properties", handlers.GetAvailableRequestProperties(applicationService))
	g.POST("", handlers.CreateRequest(applicationService))
	g.POST("/:id/set-resolution", handlers.DecideRequest(applicationService))
	g.POST("/:id/submit-expenses", handlers.SubmitRequestExpense(applicationService))
	g.PATCH("/:id/expense", handlers.SubmitRequestExpense(applicationService))
	g.POST("/:id/confirm-tenant-expenses", handlers.ConfirmTenantExpenses(applicationService))
	g.POST("/:id/complete-owner", handlers.CompleteOwnerResolution(applicationService))
	g.POST("/:id/complete-owner-request", handlers.CompleteOwnerRequest(applicationService))
}

func favoriteRoutes(g *gin.RouterGroup, favService *services.FavoritesService, jwtSecret string) {
	g.Use(middleware.Auth(jwtSecret))
	g.GET("", handlers.GetFavorites(favService))
	g.POST("/:propertyId", handlers.AddFavorite(favService))
	g.DELETE("/:propertyId", handlers.RemoveFavorite(favService))
}

// В chatRoutes везде нужен JWT, а доступ есть только у seller_id/buyer_id (подробности в handlers + services/chat).
func chatRoutes(g *gin.RouterGroup, chatService *services.ChatService, contractService *services.ContractService, jwtSecret string) {
	g.Use(middleware.Auth(jwtSecret))
	g.POST("", handlers.CreateChat(chatService))
	g.GET("", handlers.ListChats(chatService))
	// Сначала добавляем маршруты с суффиксом, чтобы их не перехватывал общий GET /:id.
	g.GET("/:id/contract-draft", handlers.GetContractDraft(contractService))
	g.GET("/:id/messages", handlers.GetChatMessages(chatService))
	g.POST("/:id/contracts", handlers.CreateChatContract(contractService))
	g.PATCH("/:id/read", handlers.MarkChatRead(chatService))
	g.POST("/:id/messages", handlers.SendChatMessage(chatService))
	g.GET("/:id", handlers.GetChat(chatService))
}

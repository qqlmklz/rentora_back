package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"rentora/backend/internal/config"
	"rentora/backend/internal/repository"
	"rentora/backend/internal/routes"
	"rentora/backend/internal/services"
	aiSvc "rentora/backend/internal/services/ai"
	"rentora/backend/internal/ws"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	if cfg.DatabaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}
	if cfg.JWTSecret == "" {
		log.Fatal("JWT_SECRET is required")
	}
	analyzerMode := strings.ToLower(strings.TrimSpace(os.Getenv("PRIORITY_ANALYZER_MODE")))
	if analyzerMode == "" {
		analyzerMode = "ai"
	}

	ctx := context.Background()
	db, err := repository.NewDB(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database: %v\nПроверьте DATABASE_URL в .env (пользователь, пароль, хост, база rentora).", err)
	}
	defer db.Close()

	db.LogApplicationsSnapshot(ctx)

	authService := services.NewAuthService(db, cfg.JWTSecret)
	profileService := services.NewProfileService(db)
	propertyService := services.NewPropertyService(db)
	var priorityAnalyzer aiSvc.PriorityAnalyzer
	switch analyzerMode {
	case "ai":
		aiAnalyzer, err := aiSvc.NewOpenAIPriorityAnalyzerFromEnv()
		if err != nil {
			log.Printf("AI priority analyzer mode=ai init_error=%v -> fallback=mock", err)
			priorityAnalyzer = aiSvc.NewMockPriorityAnalyzer()
			analyzerMode = "mock"
		} else {
			priorityAnalyzer = aiAnalyzer
		}
	default:
		priorityAnalyzer = aiSvc.NewMockPriorityAnalyzer()
		analyzerMode = "mock"
	}
	log.Printf("AI priority analyzer mode=%s", analyzerMode)
	applicationService := services.NewApplicationService(db, priorityAnalyzer)
	favoritesService := services.NewFavoritesService(db)
	hub := ws.NewHub()
	chatService := services.NewChatService(db, hub)
	contractService := services.NewContractService(db, hub)

	gin.SetMode(cfg.GinMode)
	r := gin.New()

	routes.Setup(r, cfg.CORSOrigins, authService, profileService, propertyService, applicationService, favoritesService, chatService, contractService, hub, cfg.JWTSecret)

	addr := fmt.Sprintf(":%d", cfg.Port)
	log.Printf("Rentora backend starting on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("server run: %v", err)
	}
}

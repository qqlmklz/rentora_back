package main

import (
	"context"
	"fmt"
	"log"

	"rentora/backend/internal/config"
	"rentora/backend/internal/repository"
	"rentora/backend/internal/routes"
	"rentora/backend/internal/services"

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

	ctx := context.Background()
	db, err := repository.NewDB(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database: %v\nПроверьте DATABASE_URL в .env (пользователь, пароль, хост, база rentora).", err)
	}
	defer db.Close()

	authService := services.NewAuthService(db, cfg.JWTSecret)
	profileService := services.NewProfileService(db)
	propertyService := services.NewPropertyService(db)
	favoritesService := services.NewFavoritesService(db)

	gin.SetMode(cfg.GinMode)
	r := gin.New()

	routes.Setup(r, cfg.CORSOrigins, authService, profileService, propertyService, favoritesService, cfg.JWTSecret)

	addr := fmt.Sprintf(":%d", cfg.Port)
	log.Printf("Rentora backend starting on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("server run: %v", err)
	}
}

package config

import (
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Тут держим конфиг приложения, который читаем из переменных окружения.
type Config struct {
	Port        int
	GinMode     string
	CORSOrigins []string
	DatabaseURL string
	JWTSecret   string
}

// Тут грузим .env и собираем Config, чтобы потом не дергать os.Getenv по всему проекту.
func Load() (*Config, error) {
	_ = godotenv.Load()

	port := 8080
	if p := os.Getenv("PORT"); p != "" {
		if v, err := strconv.Atoi(p); err == nil {
			port = v
		}
	}

	ginMode := os.Getenv("GIN_MODE")
	if ginMode == "" {
		ginMode = "debug"
	}

	corsOrigins := []string{"*"}
	if o := os.Getenv("CORS_ORIGINS"); o != "" {
		corsOrigins = strings.Split(o, ",")
		for i := range corsOrigins {
			corsOrigins[i] = strings.TrimSpace(corsOrigins[i])
		}
	}

	databaseURL := os.Getenv("DATABASE_URL")
	jwtSecret := os.Getenv("JWT_SECRET")

	return &Config{
		Port:        port,
		GinMode:     ginMode,
		CORSOrigins: corsOrigins,
		DatabaseURL: databaseURL,
		JWTSecret:   jwtSecret,
	}, nil
}

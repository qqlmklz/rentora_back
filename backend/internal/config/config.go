package config

import (
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds application configuration loaded from environment.
type Config struct {
	Port        int
	GinMode     string
	CORSOrigins []string
	DatabaseURL string
	JWTSecret   string
}

// Load reads .env and populates Config.
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

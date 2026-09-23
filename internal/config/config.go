package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port      string
	Env       string
	DB_URL    string
	JWTSecret string
}

func MustLoad() Config {
	godotenv.Load()

	port := os.Getenv("PORT")
	if port == "" {
		panic("PORT is not set")
	}

	env := os.Getenv("ENV")
	if env == "" {
		panic("ENV is not set")
	}

	db_url := os.Getenv("DB_URL")
	if db_url == "" {
		panic("DB_URL is not set")
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		panic("JWT_SECRET is not set")
	}

	return Config{
		Port:      port,
		Env:       env,
		DB_URL:    db_url,
		JWTSecret: jwtSecret,
	}
}

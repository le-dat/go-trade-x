package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	AppPort        string
	DatabaseURL    string
	KafkaBrokers   string
	RedisURL       string
	JWTSecret      string
	AllowedOrigins string
	MaxConnections int
}

func Load() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using defaults and system environment variables")
	}

	jwtSecret := getEnv("JWT_SECRET", "")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET environment variable is required")
	}

	return &Config{
		AppPort:        getEnv("APP_PORT", "8080"),
		DatabaseURL:    getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/gotradex?sslmode=disable"),
		KafkaBrokers:   getEnv("KAFKA_BROKERS", "localhost:9092"),
		RedisURL:       getEnv("REDIS_URL", "localhost:6379"),
		JWTSecret:      jwtSecret,
		AllowedOrigins: getEnv("ALLOWED_ORIGINS", "*"),
		MaxConnections: parseIntEnv("MAX_CONNECTIONS", 1000),
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func parseIntEnv(key string, fallback int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return fallback
}


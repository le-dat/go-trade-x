package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	AppPort      string
	DatabaseURL  string
	KafkaBrokers string
	RedisURL     string
	JWTSecret    string
}

func Load() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using defaults and system environment variables")
	}

	return &Config{
		AppPort:      getEnv("APP_PORT", "8080"),
		DatabaseURL:  getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/gotradex?sslmode=disable"),
		KafkaBrokers: getEnv("KAFKA_BROKERS", "localhost:9092"),
		RedisURL:     getEnv("REDIS_URL", "localhost:6379"),
		JWTSecret:    getEnv("JWT_SECRET", "super-secret-key"),
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}


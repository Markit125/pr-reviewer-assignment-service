package config

import (
	"log"
	"os"
)

type Config struct {
	ServerPort  string
	DatabaseDSN string
}

func LoadConfig() Config {
	port := getEnv("PORT", "8080")
	dbDSN := getEnv("DATABASE_DSN", "postgres://user:password@localhost:5432/pr_reviewer_db?sslmode=disable")

	return Config{
		ServerPort:  ":" + port,
		DatabaseDSN: dbDSN,
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	log.Printf("Using default value for %s: %s", key, fallback)
	return fallback
}

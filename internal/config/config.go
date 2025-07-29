package config

import (
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	LogLevel                string
	GeminiAPIKeys           []string
	GeminiAPIURL            string
	ValkeyHost              string
	ValkeyPort              string
	ValkeyPassword          string
	RateLimitPerMinute      int
	RateLimitWindow         string
	ProxyAPIKey             string
	AllowGeminiKeysForAuth  bool
}

func LoadConfig() *Config {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found, using environment variables")
	}

	rateLimit, err := strconv.Atoi(getEnv("RATE_LIMIT_PER_MINUTE", "60"))
	if err != nil {
		log.Fatalf("Invalid RATE_LIMIT_PER_MINUTE: %v", err)
	}

	allowAuth, err := strconv.ParseBool(getEnv("ALLOW_GEMINI_KEYS_FOR_AUTH", "true"))
	if err != nil {
		log.Fatalf("Invalid ALLOW_GEMINI_KEYS_FOR_AUTH: %v", err)
	}

	return &Config{
		LogLevel:                getEnv("LOG_LEVEL", "INFO"),
		GeminiAPIKeys:           strings.Split(getEnv("GEMINI_API_KEYS", ""), ","),
		GeminiAPIURL:            getEnv("GEMINI_API_URL", "https://generativelanguage.googleapis.com"),
		ValkeyHost:              getEnv("VALKEY_HOST", "localhost"),
		ValkeyPort:              getEnv("VALKEY_PORT", "6379"),
		ValkeyPassword:          getEnv("VALKEY_PASSWORD", ""),
		RateLimitPerMinute:      rateLimit,
		RateLimitWindow:         getEnv("RATE_LIMIT_WINDOW", "fixed"),
		ProxyAPIKey:             getEnv("PROXY_API_KEY", ""),
		AllowGeminiKeysForAuth:  allowAuth,
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
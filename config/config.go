package config

import (
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	HTTPPort                  string
	RedisHost                 string
	RedisPort                 string
	RedisPassword             string
	RedisDB                   int
	IPMaxRequests             int
	IPBlockDuration           time.Duration
	TokenMaxRequests          int
	TokenBlockDuration        time.Duration
	CustomTokenLimits         map[string]int
	CustomTokenBlockDurations map[string]time.Duration
}

func LoadConfig() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		HTTPPort:                  getEnv("HTTP_PORT", "8080"),
		RedisHost:                 getEnv("REDIS_HOST", "localhost"),
		RedisPort:                 getEnv("REDIS_PORT", "6379"),
		RedisPassword:             getEnv("REDIS_PASSWORD", ""),
		RedisDB:                   getEnvAsInt("REDIS_DB", 0),
		IPMaxRequests:             getEnvAsInt("RATE_LIMIT_IP_MAX_REQUESTS", 5),
		IPBlockDuration:           time.Duration(getEnvAsInt("RATE_LIMIT_IP_BLOCK_DURATION_SECONDS", 300)) * time.Second,
		TokenMaxRequests:          getEnvAsInt("RATE_LIMIT_TOKEN_MAX_REQUESTS", 10),
		TokenBlockDuration:        time.Duration(getEnvAsInt("RATE_LIMIT_TOKEN_BLOCK_DURATION_SECONDS", 300)) * time.Second,
		CustomTokenLimits:         make(map[string]int),
		CustomTokenBlockDurations: make(map[string]time.Duration),
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		return val
	}
	return fallback
}

func getEnvAsInt(key string, fallback int) int {
	valStr := getEnv(key, "")
	if val, err := strconv.Atoi(valStr); err == nil {
		return val
	}
	return fallback
}

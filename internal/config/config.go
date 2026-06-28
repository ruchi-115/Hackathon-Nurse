// Package config loads runtime configuration from the environment with sensible
// defaults so the pipeline runs with zero setup for the hackathon demo.
package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds all tunable settings for the pipeline and API server.
type Config struct {
	BaseURL        string        // mock PCC API base URL
	DBPath         string        // SQLite file path
	Concurrency    int           // bounded worker-pool size for fan-out fetch
	RatePerSecond  float64       // shared rate limiter: requests per second
	RateBurst      int           // shared rate limiter: burst size
	MaxRetries     int           // max retry attempts per request (429/5xx)
	HTTPTimeout    time.Duration // per-request HTTP timeout
	Port           string        // REST API listen port
	CORSOrigin     string        // allowed CORS origin for the frontend
	AnthropicKey   string        // ANTHROPIC_API_KEY; empty disables LLM fallback
	AnthropicModel string        // model id for LLM extraction
}

// Facilities are the three mock facilities exposed by the API.
var Facilities = []int{101, 102, 103}

// Load reads configuration from the environment, applying defaults.
func Load() Config {
	return Config{
		BaseURL:        getEnv("PCC_BASE_URL", "https://hackathon.prod.pulsefoundry.ai"),
		DBPath:         getEnv("DB_PATH", "./pcc.db"),
		Concurrency:    getEnvInt("CONCURRENCY", 8),
		RatePerSecond:  getEnvFloat("RATE_PER_SECOND", 12),
		RateBurst:      getEnvInt("RATE_BURST", 4),
		MaxRetries:     getEnvInt("MAX_RETRIES", 6),
		HTTPTimeout:    time.Duration(getEnvInt("HTTP_TIMEOUT_SECONDS", 30)) * time.Second,
		Port:           getEnv("PORT", "8080"),
		CORSOrigin:     getEnv("CORS_ORIGIN", "*"),
		AnthropicKey:   os.Getenv("ANTHROPIC_API_KEY"),
		AnthropicModel: getEnv("MODEL", "claude-opus-4-8"),
	}
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getEnvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func getEnvFloat(key string, def float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return def
}

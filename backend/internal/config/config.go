package config

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultPort            = "8080"
	defaultTokenTTLMinutes = 120
	defaultLLMBaseURL      = "https://llm.nitec.kz"
	defaultLLMModel        = "openai/gpt-oss-120b"
	defaultLLMTimeout      = 60
)

type Config struct {
	Port           string
	LiveKitURL     string
	LiveKitAPIKey  string
	LiveKitSecret  string
	TokenTTL       time.Duration
	AllowedOrigins []string
	LLMBaseURL     string
	LLMAPIKey      string
	LLMModel       string
	LLMTimeout     time.Duration
}

func Load() Config {
	loadDotEnv(".env")

	ttlMinutes := envInt("TOKEN_TTL_MINUTES", defaultTokenTTLMinutes)

	return Config{
		Port:           env("PORT", defaultPort),
		LiveKitURL:     strings.TrimSpace(os.Getenv("LIVEKIT_URL")),
		LiveKitAPIKey:  strings.TrimSpace(os.Getenv("LIVEKIT_API_KEY")),
		LiveKitSecret:  strings.TrimSpace(os.Getenv("LIVEKIT_API_SECRET")),
		TokenTTL:       time.Duration(ttlMinutes) * time.Minute,
		AllowedOrigins: splitCSV(env("ALLOWED_ORIGINS", "http://localhost:5173")),
		LLMBaseURL:     strings.TrimRight(env("LLM_BASE_URL", defaultLLMBaseURL), "/"),
		LLMAPIKey:      strings.TrimSpace(os.Getenv("LLM_API_KEY")),
		LLMModel:       env("LLM_MODEL", defaultLLMModel),
		LLMTimeout:     time.Duration(envInt("LLM_TIMEOUT_SECONDS", defaultLLMTimeout)) * time.Second,
	}
}

func env(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func envInt(key string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}

	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return fallback
	}

	return value
}

func splitCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	values := make([]string, 0, len(parts))

	for _, part := range parts {
		value := strings.TrimSpace(part)
		if value != "" {
			values = append(values, value)
		}
	}

	return values
}

func loadDotEnv(path string) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}

		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		value = strings.Trim(value, `"'`)

		if key == "" || os.Getenv(key) != "" {
			continue
		}

		_ = os.Setenv(key, value)
	}
}

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
	defaultLLMModel        = "moonshotai/Kimi-K2.6"
	defaultSTTModel        = "openai/whisper-large-v3"
	defaultLLMTimeout      = 60
	defaultSTTTimeout      = 900
	defaultReportsStorage  = "data/reports.json"
	defaultRecordingsDir   = "data/recordings"
)

type Config struct {
	Port                       string
	LiveKitURL                 string
	LiveKitAPIKey              string
	LiveKitSecret              string
	LiveKitEgressEnabled       bool
	LiveKitEgressAudioOnly     bool
	LiveKitEgressLayout        string
	LiveKitEgressFilePrefix    string
	LiveKitEgressPublicBaseURL string
	LiveKitEgressWebhookURL    string
	LiveKitS3AccessKey         string
	LiveKitS3Secret            string
	LiveKitS3SessionToken      string
	LiveKitS3Region            string
	LiveKitS3Endpoint          string
	LiveKitS3Bucket            string
	LiveKitS3ForcePathStyle    bool
	TokenTTL                   time.Duration
	AllowedOrigins             []string
	KeycloakIssuerURL          string
	KeycloakJWKSURL            string
	KeycloakTokenURL           string
	KeycloakClientID           string
	LLMBaseURL                 string
	LLMAPIKey                  string
	LLMModel                   string
	STTBaseURL                 string
	STTAPIKey                  string
	STTModel                   string
	LLMTimeout                 time.Duration
	STTTimeout                 time.Duration
	ReportsStoragePath         string
	RecordingsStoragePath      string
}

func Load() Config {
	loadDotEnv(".env")

	ttlMinutes := envInt("TOKEN_TTL_MINUTES", defaultTokenTTLMinutes)

	return Config{
		Port:                       env("PORT", defaultPort),
		LiveKitURL:                 strings.TrimSpace(os.Getenv("LIVEKIT_URL")),
		LiveKitAPIKey:              strings.TrimSpace(os.Getenv("LIVEKIT_API_KEY")),
		LiveKitSecret:              strings.TrimSpace(os.Getenv("LIVEKIT_API_SECRET")),
		LiveKitEgressEnabled:       envBool("LIVEKIT_EGRESS_ENABLED", false),
		LiveKitEgressAudioOnly:     envBool("LIVEKIT_EGRESS_AUDIO_ONLY", true),
		LiveKitEgressLayout:        env("LIVEKIT_EGRESS_LAYOUT", "grid"),
		LiveKitEgressFilePrefix:    env("LIVEKIT_EGRESS_FILE_PREFIX", "alemlive"),
		LiveKitEgressPublicBaseURL: strings.TrimRight(strings.TrimSpace(os.Getenv("LIVEKIT_EGRESS_PUBLIC_BASE_URL")), "/"),
		LiveKitEgressWebhookURL:    strings.TrimSpace(os.Getenv("LIVEKIT_EGRESS_WEBHOOK_URL")),
		LiveKitS3AccessKey:         strings.TrimSpace(os.Getenv("LIVEKIT_S3_ACCESS_KEY")),
		LiveKitS3Secret:            strings.TrimSpace(os.Getenv("LIVEKIT_S3_SECRET")),
		LiveKitS3SessionToken:      strings.TrimSpace(os.Getenv("LIVEKIT_S3_SESSION_TOKEN")),
		LiveKitS3Region:            strings.TrimSpace(os.Getenv("LIVEKIT_S3_REGION")),
		LiveKitS3Endpoint:          strings.TrimSpace(os.Getenv("LIVEKIT_S3_ENDPOINT")),
		LiveKitS3Bucket:            strings.TrimSpace(os.Getenv("LIVEKIT_S3_BUCKET")),
		LiveKitS3ForcePathStyle:    envBool("LIVEKIT_S3_FORCE_PATH_STYLE", false),
		TokenTTL:                   time.Duration(ttlMinutes) * time.Minute,
		AllowedOrigins:             splitCSV(env("ALLOWED_ORIGINS", "http://localhost:5173")),
		KeycloakIssuerURL:          strings.TrimRight(strings.TrimSpace(os.Getenv("KEYCLOAK_ISSUER_URL")), "/"),
		KeycloakJWKSURL:            strings.TrimSpace(os.Getenv("KEYCLOAK_JWKS_URL")),
		KeycloakTokenURL:           strings.TrimSpace(os.Getenv("KEYCLOAK_TOKEN_URL")),
		KeycloakClientID:           strings.TrimSpace(os.Getenv("KEYCLOAK_CLIENT_ID")),
		LLMBaseURL:                 strings.TrimRight(env("LLM_BASE_URL", defaultLLMBaseURL), "/"),
		LLMAPIKey:                  strings.TrimSpace(os.Getenv("LLM_API_KEY")),
		LLMModel:                   env("LLM_MODEL", defaultLLMModel),
		STTBaseURL:                 strings.TrimRight(env("STT_BASE_URL", env("LLM_BASE_URL", defaultLLMBaseURL)), "/"),
		STTAPIKey:                  strings.TrimSpace(env("STT_API_KEY", os.Getenv("LLM_API_KEY"))),
		STTModel:                   env("STT_MODEL", defaultSTTModel),
		LLMTimeout:                 time.Duration(envInt("LLM_TIMEOUT_SECONDS", defaultLLMTimeout)) * time.Second,
		STTTimeout:                 time.Duration(envInt("STT_TIMEOUT_SECONDS", defaultSTTTimeout)) * time.Second,
		ReportsStoragePath:         strings.TrimSpace(env("REPORTS_STORAGE_PATH", defaultReportsStorage)),
		RecordingsStoragePath:      strings.TrimSpace(env("RECORDINGS_STORAGE_PATH", defaultRecordingsDir)),
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

func envBool(key string, fallback bool) bool {
	raw := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	if raw == "" {
		return fallback
	}
	switch raw {
	case "1", "true", "yes", "y", "on":
		return true
	case "0", "false", "no", "n", "off":
		return false
	default:
		return fallback
	}
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

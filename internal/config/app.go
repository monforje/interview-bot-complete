package config

import (
	"os"
	"strconv"
	"time"
)

type AppConfig struct {
	OpenAI   OpenAIConfig
	Telegram TelegramConfig
	Server   ServerConfig
}

type TelegramConfig struct {
	Token      string
	WebhookURL string
	Debug      bool
}

type ServerConfig struct {
	Port            int
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
}

func LoadAppConfig() *AppConfig {
	return &AppConfig{
		OpenAI: OpenAIConfig{
			APIKey:      getEnv("OPENAI_API_KEY", ""),
			Model:       getEnv("OPENAI_MODEL", "gpt-4"),
			MaxTokens:   getEnvAsInt("OPENAI_MAX_TOKENS", 4000),
			Temperature: getEnvAsFloat("OPENAI_TEMPERATURE", 0.1),
		},
		Telegram: TelegramConfig{
			Token:      getEnv("TELEGRAM_BOT_TOKEN", ""),
			WebhookURL: getEnv("TELEGRAM_WEBHOOK_URL", ""),
			Debug:      getEnvAsBool("TELEGRAM_DEBUG", false),
		},
		Server: ServerConfig{
			Port:            getEnvAsInt("SERVER_PORT", 8080),
			ReadTimeout:     getEnvAsDuration("SERVER_READ_TIMEOUT", 10*time.Second),
			WriteTimeout:    getEnvAsDuration("SERVER_WRITE_TIMEOUT", 10*time.Second),
			ShutdownTimeout: getEnvAsDuration("SERVER_SHUTDOWN_TIMEOUT", 30*time.Second),
		},
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getEnvAsFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
			return floatValue
		}
	}
	return defaultValue
}

func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

package config

import (
	"fmt"
	"os"
	"strconv"
)

type OpenAIConfig struct {
	APIKey      string
	Model       string
	MaxTokens   int
	Temperature float64
}

// LoadOpenAIConfig загружает конфигурацию OpenAI из переменных окружения
func LoadOpenAIConfig() *OpenAIConfig {
	config := &OpenAIConfig{
		APIKey:      os.Getenv("OPENAI_API_KEY"),
		Model:       getEnvOrDefault("OPENAI_MODEL", "gpt-4o"),
		MaxTokens:   getEnvAsIntOrDefault("OPENAI_MAX_TOKENS", 4000),
		Temperature: getEnvAsFloatOrDefault("OPENAI_TEMPERATURE", 0.1),
	}

	return config
}

// ValidateConfig проверяет корректность конфигурации
func (c *OpenAIConfig) ValidateConfig() error {
	if c.APIKey == "" {
		return fmt.Errorf("OPENAI_API_KEY is required")
	}

	if c.MaxTokens <= 0 {
		return fmt.Errorf("OPENAI_MAX_TOKENS must be positive")
	}

	if c.Temperature < 0 || c.Temperature > 2 {
		return fmt.Errorf("OPENAI_TEMPERATURE must be between 0 and 2")
	}

	return nil
}

// GetModelInfo возвращает информацию о используемой модели
func (c *OpenAIConfig) GetModelInfo() map[string]interface{} {
	return map[string]interface{}{
		"model":       c.Model,
		"max_tokens":  c.MaxTokens,
		"temperature": c.Temperature,
		"provider":    "OpenAI",
	}
}

// helper функции для чтения переменных окружения
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsIntOrDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvAsFloatOrDefault(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
			return floatValue
		}
	}
	return defaultValue
}

package api

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

type OpenAIClient struct {
	apiKey      string
	model       string
	maxTokens   int
	temperature float64
	client      *http.Client
	logger      *slog.Logger
}

type OpenAIRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature"`
	MaxTokens   int       `json:"max_tokens"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OpenAIResponse struct {
	ID      string    `json:"id"`
	Object  string    `json:"object"`
	Created int64     `json:"created"`
	Model   string    `json:"model"`
	Choices []Choice  `json:"choices"`
	Usage   Usage     `json:"usage"`
	Error   *APIError `json:"error,omitempty"`
}

type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type APIError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code"`
}

func NewOpenAIClient(apiKey string) *OpenAIClient {
	// Читаем настройки из переменных окружения
	model := getEnvOrDefault("OPENAI_MODEL", "gpt-4.1-mini")
	maxTokens := getEnvAsIntOrDefault("OPENAI_MAX_TOKENS", 4000)
	temperature := getEnvAsFloatOrDefault("OPENAI_TEMPERATURE", 0.1)

	// Настройка транспорта для лучшей производительности
	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxConnsPerHost:     100,
		MaxIdleConnsPerHost: 100,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: false,
		},
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
	}

	return &OpenAIClient{
		apiKey:      apiKey,
		model:       model,
		maxTokens:   maxTokens,
		temperature: temperature,
		client: &http.Client{
			Timeout:   120 * time.Second,
			Transport: transport,
		},
		logger: slog.Default(),
	}
}

// ExtractProfile - единственный метод для работы с профилями
func (c *OpenAIClient) ExtractProfile(prompt string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	reqBody := OpenAIRequest{
		Model: c.model,
		Messages: []Message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Temperature: c.temperature,
		MaxTokens:   c.maxTokens,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		c.logger.Error("Failed to marshal request", "error", err)
		return "", fmt.Errorf("error marshaling request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonBody))
	if err != nil {
		c.logger.Error("Failed to create request", "error", err)
		return "", fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		c.logger.Error("Failed to make request", "error", err)
		return "", fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.logger.Error("Failed to read response", "error", err)
		return "", fmt.Errorf("error reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		c.logger.Error("OpenAI API error", "status", resp.StatusCode, "body", string(body))
		return "", fmt.Errorf("OpenAI API error: status %d, body: %s", resp.StatusCode, string(body))
	}

	var openAIResp OpenAIResponse
	if err := json.Unmarshal(body, &openAIResp); err != nil {
		c.logger.Error("Failed to unmarshal response", "error", err)
		return "", fmt.Errorf("error unmarshaling response: %w", err)
	}

	if openAIResp.Error != nil {
		c.logger.Error("OpenAI API returned error", "error", openAIResp.Error.Message)
		return "", fmt.Errorf("OpenAI API error: %s", openAIResp.Error.Message)
	}

	if len(openAIResp.Choices) == 0 {
		c.logger.Error("No choices returned from OpenAI API")
		return "", fmt.Errorf("no choices returned from OpenAI API")
	}

	content := openAIResp.Choices[0].Message.Content
	content = cleanJSONResponse(content)

	// Логируем использование токенов
	if openAIResp.Usage.TotalTokens > 0 {
		c.logger.Info("Token usage",
			"prompt_tokens", openAIResp.Usage.PromptTokens,
			"completion_tokens", openAIResp.Usage.CompletionTokens,
			"total_tokens", openAIResp.Usage.TotalTokens)
	}

	c.logger.Info("Successfully extracted profile", "content_length", len(content))
	return content, nil
}

// cleanJSONResponse удаляет markdown форматирование из ответа
func cleanJSONResponse(response string) string {
	// Удаляем ```json и ``` блоки
	response = strings.ReplaceAll(response, "```json", "")
	response = strings.ReplaceAll(response, "```", "")

	// Убираем лишние пробелы и переносы строк в начале и конце
	response = strings.TrimSpace(response)

	return response
}

// Helper функции для работы с переменными окружения
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsIntOrDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := parseIntSafe(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvAsFloatOrDefault(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatValue, err := parseFloatSafe(value); err == nil {
			return floatValue
		}
	}
	return defaultValue
}

func parseIntSafe(s string) (int, error) {
	// Простая реализация без импорта strconv для избежания конфликтов
	var result int
	_, err := fmt.Sscanf(s, "%d", &result)
	return result, err
}

func parseFloatSafe(s string) (float64, error) {
	// Простая реализация без импорта strconv для избежания конфликтов
	var result float64
	_, err := fmt.Sscanf(s, "%f", &result)
	return result, err
}

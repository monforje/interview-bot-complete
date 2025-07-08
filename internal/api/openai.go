package api

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"
)

type OpenAIClient struct {
	apiKey string
	client *http.Client
	logger *slog.Logger
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
		apiKey: apiKey,
		client: &http.Client{
			Timeout:   120 * time.Second,
			Transport: transport,
		},
		logger: slog.Default(),
	}
}

// NewOpenAIClientWithConfig создает клиент с расширенной конфигурацией
func NewOpenAIClientWithConfig(apiKey, model string, maxTokens int, temperature float64) *OpenAIClient {
	client := &OpenAIClient{
		apiKey: apiKey,
		client: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
	return client
}

func (c *OpenAIClient) ExtractProfile(prompt string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	reqBody := OpenAIRequest{
		Model: "gpt-4", // Обновить на более стабильную модель
		Messages: []Message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Temperature: 0.1,
		MaxTokens:   4000,
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

// GetUsageStats возвращает статистику использования токенов (доступно только в OpenAI)
func (c *OpenAIClient) GetUsageStats(prompt string) (*Usage, error) {
	reqBody := OpenAIRequest{
		Model: "gpt-4o",
		Messages: []Message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Temperature: 0.1,
		MaxTokens:   4000,
	}

	jsonBody, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	var openAIResp OpenAIResponse
	json.Unmarshal(body, &openAIResp)

	return &openAIResp.Usage, nil
}

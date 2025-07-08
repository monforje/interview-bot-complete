package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

type OpenAIClient struct {
	apiKey string
	client *http.Client
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
	return &OpenAIClient{
		apiKey: apiKey,
		client: &http.Client{
			Timeout: 120 * time.Second, // Увеличенный таймаут для сложных запросов
		},
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
	reqBody := OpenAIRequest{
		Model: "gpt-4o", // Используем самую современную модель GPT-4o
		Messages: []Message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Temperature: 0.1,  // Низкая температура для точности
		MaxTokens:   4000, // Увеличен лимит для более детальных профилей
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("error marshaling request: %w", err)
	}

	// Создание запроса к официальному OpenAI API
	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("error creating request: %w", err)
	}

	// Установка заголовков для OpenAI API
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("OpenAI API error: status %d, body: %s", resp.StatusCode, string(body))
	}

	var openAIResp OpenAIResponse
	if err := json.Unmarshal(body, &openAIResp); err != nil {
		return "", fmt.Errorf("error unmarshaling response: %w", err)
	}

	if openAIResp.Error != nil {
		return "", fmt.Errorf("OpenAI API error: %s", openAIResp.Error.Message)
	}

	if len(openAIResp.Choices) == 0 {
		return "", fmt.Errorf("no choices returned from OpenAI API")
	}

	content := openAIResp.Choices[0].Message.Content

	// Очистка от markdown форматирования
	content = cleanJSONResponse(content)

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

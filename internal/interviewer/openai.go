package interviewer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"interview-bot-complete/internal/config"
	"io"
	"net/http"
	"os"
)

// OpenAI API структуры
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
	Choices []Choice  `json:"choices"`
	Error   *APIError `json:"error,omitempty"`
}

type Choice struct {
	Message Message `json:"message"`
}

type APIError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
}

const openaiURL = "https://api.openai.com/v1/chat/completions"

// getModelFromEnv возвращает модель из переменных окружения
func getModelFromEnv() string {
	model := os.Getenv("OPENAI_MODEL")
	if model == "" {
		return "gpt-4.1-mini" // значение по умолчанию
	}
	return model
}

// callOpenAI делает запрос к OpenAI API
func (s *Service) callOpenAI(messages []Message, cfg *config.Config) (string, error) {
	// Получаем модель из переменных окружения
	model := getModelFromEnv()

	// Динамически рассчитываем max_tokens на основе конфигурации
	maxTokens := 500 + (cfg.GetQuestionsPerBlock()+cfg.GetMaxFollowupQuestions())*100

	// Подготавливаем запрос
	request := OpenAIRequest{
		Model:       model,
		Messages:    messages,
		Temperature: 0.7,
		MaxTokens:   maxTokens,
	}

	// Сериализуем в JSON
	jsonData, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("ошибка сериализации запроса: %w", err)
	}

	// Создаем HTTP запрос
	req, err := http.NewRequest("POST", openaiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("ошибка создания запроса: %w", err)
	}

	// Устанавливаем заголовки
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.apiKey))

	// Выполняем запрос
	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("ошибка выполнения запроса: %w", err)
	}
	defer resp.Body.Close()

	// Читаем ответ
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("ошибка чтения ответа: %w", err)
	}

	// Проверяем статус код
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP ошибка %d: %s", resp.StatusCode, string(body))
	}

	// Парсим ответ
	var openaiResp OpenAIResponse
	err = json.Unmarshal(body, &openaiResp)
	if err != nil {
		return "", fmt.Errorf("ошибка парсинга ответа: %w", err)
	}

	// Проверяем на ошибки API
	if openaiResp.Error != nil {
		return "", fmt.Errorf("OpenAI API ошибка: %s", openaiResp.Error.Message)
	}

	// Проверяем наличие ответа
	if len(openaiResp.Choices) == 0 {
		return "", fmt.Errorf("пустой ответ от OpenAI")
	}

	return openaiResp.Choices[0].Message.Content, nil
}

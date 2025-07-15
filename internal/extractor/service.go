package extractor

import (
	"encoding/json"
	"fmt"
	"interview-bot-complete/internal/api"
	"interview-bot-complete/internal/interview"
	"interview-bot-complete/internal/prompts"
	"interview-bot-complete/internal/schema"
	"interview-bot-complete/internal/storage"
	"interview-bot-complete/internal/validator"
	"io/ioutil"
	"log"
	"os"
	"time"
)

// Service представляет сервис извлечения профилей
type Service struct {
	apiClient    *api.OpenAIClient
	schemaFields map[string]schema.SchemaField
}

// ProfileResult представляет результат анализа профиля
type ProfileResult struct {
	ProfileJSON string                 `json:"profile_json"`
	Metadata    map[string]interface{} `json:"metadata"`
	Success     bool                   `json:"success"`
	Error       string                 `json:"error,omitempty"`
}

// New создает новый сервис экстрактора
func New(openaiAPIKey string) (*Service, error) {
	// Создаем клиент API
	client := api.NewOpenAIClient(openaiAPIKey)

	// Загружаем схему из config/profile_schema.yaml
	yamlContent, err := ioutil.ReadFile("config/profile_schema.yaml")
	if err != nil {
		return nil, fmt.Errorf("error reading config/profile_schema.yaml: %w", err)
	}

	// Парсим схему
	schemaFields, err := schema.ParseYAMLSchema(yamlContent)
	if err != nil {
		return nil, fmt.Errorf("error parsing schema: %w", err)
	}

	log.Printf("Profile Extractor: Загружена схема с %d полями", len(schemaFields))

	return &Service{
		apiClient:    client,
		schemaFields: schemaFields,
	}, nil
}

// ExtractProfile извлекает профиль из результата интервью (оптимизированно - один запрос)
func (s *Service) ExtractProfile(interviewResult *storage.InterviewResult) (*ProfileResult, error) {
	log.Printf("Начинаю извлечение профиля для интервью: %s", interviewResult.InterviewID)

	// Конвертируем InterviewResult в формат Profile Extractor
	extractorInterview := s.convertToExtractorFormat(interviewResult)

	// Извлекаем контекстуальные ответы
	userText := extractorInterview.ExtractContextualAnswers()
	log.Printf("Извлечено текста: %d символов", len(userText))

	// ЕДИНСТВЕННЫЙ запрос к API - извлечение и валидация в одном промпте
	log.Println("Извлечение профиля (оптимизированно)...")
	optimizedPrompt := prompts.GenerateOptimizedExtractionPrompt(s.schemaFields, userText)

	profileJSON, err := s.apiClient.ExtractProfile(optimizedPrompt)
	if err != nil {
		return &ProfileResult{
			Success: false,
			Error:   fmt.Sprintf("Ошибка извлечения профиля: %v", err),
		}, err
	}

	// Быстрая проверка структуры без дополнительных запросов
	if err := validator.ValidateProfileJSON(profileJSON, s.schemaFields); err != nil {
		log.Printf("Предупреждение валидации: %v", err)
	}

	// Парсим JSON для проверки
	var formatted map[string]interface{}
	if err := json.Unmarshal([]byte(profileJSON), &formatted); err != nil {
		return &ProfileResult{
			Success: false,
			Error:   fmt.Sprintf("Ошибка парсинга JSON: %v", err),
		}, err
	}

	// Добавляем минимальные метаданные
	extractorInterview = s.convertToExtractorFormat(interviewResult)
	metadata := extractorInterview.GetInterviewMetadata()

	// Только важные метаданные
	formatted["_metadata"] = map[string]interface{}{
		"interview_id":    interviewResult.InterviewID,
		"creation_date":   time.Now().Format("2006-01-02 15:04:05"),
		"total_questions": metadata["total_questions"],
		"completion_rate": metadata["completion_rate"],
	}

	// Конвертируем обратно в JSON строку
	finalJSON, err := json.MarshalIndent(formatted, "", "  ")
	if err != nil {
		return &ProfileResult{
			Success: false,
			Error:   fmt.Sprintf("Ошибка форматирования финального JSON: %v", err),
		}, err
	}

	log.Printf("Извлечение профиля завершено успешно для интервью: %s", interviewResult.InterviewID)

	return &ProfileResult{
		ProfileJSON: string(finalJSON),
		Metadata:    metadata,
		Success:     true,
	}, nil
}

// SaveProfile сохраняет профиль в файл
func (s *Service) SaveProfile(interviewID string, profileResult *ProfileResult) (string, error) {
	// Создаем папку output если не существует
	if err := os.MkdirAll("output", 0755); err != nil {
		return "", fmt.Errorf("ошибка создания папки output: %w", err)
	}

	// Сохраняем результат с ID интервью в имени файла
	fileName := fmt.Sprintf("output/profile_%s.json", interviewID)
	err := ioutil.WriteFile(fileName, []byte(profileResult.ProfileJSON), 0644)
	if err != nil {
		return "", fmt.Errorf("ошибка сохранения профиля: %w", err)
	}

	log.Printf("Профиль сохранен в: %s", fileName)
	return fileName, nil
}

// convertToExtractorFormat конвертирует InterviewResult в формат Profile Extractor
func (s *Service) convertToExtractorFormat(result *storage.InterviewResult) *interview.Interview {
	var blocks []interview.Block

	for _, block := range result.Blocks {
		var qas []interview.QuestionAndAnswer

		for _, qa := range block.QuestionsAndAnswers {
			qas = append(qas, interview.QuestionAndAnswer{
				Question: qa.Question,
				Answer:   qa.Answer,
			})
		}

		blocks = append(blocks, interview.Block{
			BlockID:             block.BlockID,
			BlockName:           block.BlockName,
			QuestionsAndAnswers: qas,
		})
	}

	return &interview.Interview{
		InterviewID: result.InterviewID,
		Timestamp:   result.Timestamp,
		Blocks:      blocks,
	}
}

// GetProfileSummary создает краткое резюме профиля для отправки в Telegram
func (s *Service) GetProfileSummary(profileJSON string) (string, error) {
	var profile map[string]interface{}
	if err := json.Unmarshal([]byte(profileJSON), &profile); err != nil {
		return "", err
	}

	summary := "📊 **Краткое резюме профиля:**\n\n"

	// Извлекаем ключевые данные из нового формата
	if name, ok := profile["name"].(string); ok && name != "" {
		summary += fmt.Sprintf("👤 **Имя:** %s\n", name)
	}

	if university, ok := profile["university"].(string); ok && university != "" {
		summary += fmt.Sprintf("🎓 **Университет:** %s\n", university)
	}

	if position, ok := profile["current_position"].(string); ok && position != "" {
		summary += fmt.Sprintf("💼 **Позиция:** %s\n", position)
	}

	if hobbies, ok := profile["hobbies"].([]interface{}); ok && len(hobbies) > 0 {
		summary += "🎯 **Хобби:** "
		for i, hobby := range hobbies {
			if i > 0 && i < 3 {
				summary += ", "
			}
			if i >= 3 {
				summary += "..."
				break
			}
			summary += fmt.Sprintf("%v", hobby)
		}
		summary += "\n"
	}

	if skills, ok := profile["hard_skills"].([]interface{}); ok && len(skills) > 0 {
		summary += "💪 **Навыки:** "
		for i, skill := range skills {
			if i > 0 && i < 3 {
				summary += ", "
			}
			if i >= 3 {
				summary += "..."
				break
			}
			summary += fmt.Sprintf("%v", skill)
		}
		summary += "\n"
	}

	summary += "\n_Полный профиль сохранен в JSON файле._"

	return summary, nil
}

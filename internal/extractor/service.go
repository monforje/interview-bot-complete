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
)

// Service представляет сервис извлечения профилей
type Service struct {
	apiClient       *api.OpenAIClient
	schemaFields    map[string]schema.SchemaField
	lastProfileJSON map[string]string // interviewID → JSON
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
		apiClient:       client,
		schemaFields:    schemaFields,
		lastProfileJSON: make(map[string]string),
	}, nil
}

// ExtractProfile извлекает психологический профиль из результата интервью
func (s *Service) ExtractProfile(interviewResult *storage.InterviewResult) (*ProfileResult, error) {
	log.Printf("Начинаю извлечение профиля для интервью: %s", interviewResult.InterviewID)

	// Конвертируем InterviewResult в формат Profile Extractor
	extractorInterview := s.convertToExtractorFormat(interviewResult)

	// Извлекаем контекстуальные ответы
	userText := extractorInterview.ExtractContextualAnswers()
	log.Printf("Извлечено текста: %d символов", len(userText))

	// Этап 1: Извлечение данных
	log.Println("Этап 1: Извлечение данных профиля...")
	extractionPrompt := prompts.GenerateExtractionPrompt(s.schemaFields, userText)

	profileJSON, err := s.apiClient.ExtractProfile(extractionPrompt)
	if err != nil {
		return &ProfileResult{
			Success: false,
			Error:   fmt.Sprintf("Ошибка извлечения профиля: %v", err),
		}, err
	}

	// Этап 2: Валидация и очистка
	log.Println("Этап 2: Валидация и очистка профиля...")
	validationPrompt := prompts.GenerateValidationPrompt(profileJSON)

	validatedJSON, err := s.apiClient.ExtractProfile(validationPrompt)
	if err != nil {
		return &ProfileResult{
			Success: false,
			Error:   fmt.Sprintf("Ошибка валидации профиля: %v", err),
		}, err
	}

	// Финальная проверка структуры
	if err := validator.ValidateProfileJSON(validatedJSON, s.schemaFields); err != nil {
		log.Printf("Предупреждение валидации: %v", err)
	}

	// Форматирование и добавление метаданных
	var formatted map[string]interface{}
	if err := json.Unmarshal([]byte(validatedJSON), &formatted); err != nil {
		return &ProfileResult{
			Success: false,
			Error:   fmt.Sprintf("Ошибка парсинга финального JSON: %v", err),
		}, err
	}

	// Fallback-валидация: все profile_fields должны быть заполнены (не null)
	missingFields := []string{}
	for field := range s.schemaFields {
		if v, ok := formatted[field]; !ok || v == nil {
			missingFields = append(missingFields, field)
		}
	}

	attempts := 0
	for len(missingFields) > 0 && attempts < 2 {
		log.Printf("Профиль не содержит все поля. Повторная генерация. Не хватает: %v", missingFields)
		// Уточняющий промпт: "Заполни только недостающие поля: ..."
		prompt := fmt.Sprintf("Заполни только недостающие поля из списка: %v. Если данных нет — ставь null. Верни только JSON.", missingFields)
		userText := extractorInterview.ExtractContextualAnswers()
		newJSON, err := s.apiClient.ExtractProfile(prompt + "\n\nТЕКСТ:\n" + userText)
		if err != nil {
			return &ProfileResult{
				Success: false,
				Error:   fmt.Sprintf("Ошибка повторной генерации профиля: %v", err),
			}, err
		}
		var newFields map[string]interface{}
		if err := json.Unmarshal([]byte(newJSON), &newFields); err != nil {
			return &ProfileResult{
				Success: false,
				Error:   fmt.Sprintf("Ошибка парсинга повторного JSON: %v", err),
			}, err
		}
		for k, v := range newFields {
			if v != nil {
				formatted[k] = v
			}
		}
		missingFields = []string{}
		for field := range s.schemaFields {
			if v, ok := formatted[field]; !ok || v == nil {
				missingFields = append(missingFields, field)
			}
		}
		attempts++
	}

	if len(missingFields) > 0 {
		return &ProfileResult{
			Success: false,
			Error:   fmt.Sprintf("Не удалось заполнить все поля профиля: %v", missingFields),
		}, fmt.Errorf("не удалось заполнить все поля профиля: %v", missingFields)
	}

	// Добавляем метаданные интервью
	metadata := extractorInterview.GetInterviewMetadata()
	formatted["_metadata"] = map[string]interface{}{
		"source_interview": metadata,
		"processing_info": map[string]interface{}{
			"schema_version":    "1.0",
			"extraction_method": "contextual_answers",
			"text_length":       len(userText),
			"extraction_source": "telegram_bot",
		},
	}

	// Конвертируем обратно в JSON строку
	finalJSON, err := json.MarshalIndent(formatted, "", "  ")
	if err != nil {
		return &ProfileResult{
			Success: false,
			Error:   fmt.Sprintf("Ошибка форматирования финального JSON: %v", err),
		}, err
	}

	// Сохраняем под ключом interviewID
	s.lastProfileJSON[interviewResult.InterviewID] = string(finalJSON)

	log.Printf("Извлечение профиля завершено успешно для интервью: %s", interviewResult.InterviewID)

	// После валидации и парсинга профиля:
	// 1. Проверить, что все поля из s.schemaFields (profile_fields) присутствуют и не равны null.
	// 2. Если нет — повторить генерацию с уточняющим промптом (до 2 раз).

	return &ProfileResult{
		ProfileJSON: string(finalJSON),
		Metadata:    metadata,
		Success:     true,
	}, nil
}

// GetLastProfileJSON возвращает последний сохранённый профиль по ID интервью
func (s *Service) GetLastProfileJSON(interviewID string) (string, bool) {
	jsonData, ok := s.lastProfileJSON[interviewID]
	return jsonData, ok
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

	// Извлекаем ключевые данные
	if values, ok := profile["values"].(map[string]interface{}); ok {
		if coreBeliefs, ok := values["core_beliefs"].([]interface{}); ok && len(coreBeliefs) > 0 {
			summary += "🎯 **Ценности:** "
			for i, belief := range coreBeliefs {
				if i > 0 {
					summary += ", "
				}
				summary += fmt.Sprintf("%v", belief)
			}
			summary += "\n\n"
		}
	}

	if personality, ok := profile["personality"].(map[string]interface{}); ok {
		if strengths, ok := personality["strengths"].([]interface{}); ok && len(strengths) > 0 {
			summary += "💪 **Сильные стороны:** "
			for i, strength := range strengths {
				if i > 0 {
					summary += ", "
				}
				summary += fmt.Sprintf("%v", strength)
			}
			summary += "\n\n"
		}
	}

	if career, ok := profile["career"].(map[string]interface{}); ok {
		if workValues, ok := career["work_values"].([]interface{}); ok && len(workValues) > 0 {
			summary += "🏢 **Рабочие ценности:** "
			for i, value := range workValues {
				if i > 0 {
					summary += ", "
				}
				summary += fmt.Sprintf("%v", value)
			}
			summary += "\n\n"
		}
	}

	if future, ok := profile["future"].(map[string]interface{}); ok {
		if aspirations, ok := future["career_aspirations"].([]interface{}); ok && len(aspirations) > 0 {
			summary += "🚀 **Карьерные цели:** "
			limit := min(3, len(aspirations))
			for i := 0; i < limit; i++ {
				if i > 0 {
					summary += ", "
				}
				if aspMap, ok := aspirations[i].(map[string]interface{}); ok {
					summary += fmt.Sprintf("%v", aspMap["goal"])
				}
			}
			summary += "\n\n"
		}
	}

	// Добавляем метаинформацию
	if metadata, ok := profile["_metadata"].(map[string]interface{}); ok {
		if sourceInterview, ok := metadata["source_interview"].(map[string]interface{}); ok {
			if completionRate, ok := sourceInterview["completion_rate"].(float64); ok {
				summary += fmt.Sprintf("📈 **Полнота интервью:** %.1f%%\n", completionRate)
			}
		}
	}

	summary += "\n_Это автоматически сгенерированный анализ на основе ваших ответов в интервью._"

	return summary, nil
}

// min helper function
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (s *Service) InferProfileMatch(profileJSON string) (*ProfileMatch, error) {
	prompt := prompts.GenerateProfileMatchPrompt(profileJSON)

	result, err := s.apiClient.ExtractProfile(prompt)
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса к OpenAI: %w", err)
	}

	var match ProfileMatch
	if err := json.Unmarshal([]byte(result), &match); err != nil {
		return nil, fmt.Errorf("ошибка парсинга ответа: %w", err)
	}

	return &match, nil
}

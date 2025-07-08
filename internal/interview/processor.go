package interview

import (
	"encoding/json"
	"fmt"
	"strings"
)

type Interview struct {
	InterviewID string  `json:"interview_id"`
	Timestamp   string  `json:"timestamp"`
	Blocks      []Block `json:"blocks"`
}

type Block struct {
	BlockID             int                 `json:"block_id"`
	BlockName           string              `json:"block_name"`
	QuestionsAndAnswers []QuestionAndAnswer `json:"questions_and_answers"`
}

type QuestionAndAnswer struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

// ParseInterviewJSON парсит JSON файл интервью
func ParseInterviewJSON(jsonData []byte) (*Interview, error) {
	var interview Interview
	err := json.Unmarshal(jsonData, &interview)
	if err != nil {
		return nil, fmt.Errorf("error parsing interview JSON: %w", err)
	}
	return &interview, nil
}

// ExtractAllAnswers извлекает все ответы из интервью в один текст
func (i *Interview) ExtractAllAnswers() string {
	var answers []string

	for _, block := range i.Blocks {
		for _, qa := range block.QuestionsAndAnswers {
			if strings.TrimSpace(qa.Answer) != "" {
				answers = append(answers, qa.Answer)
			}
		}
	}

	return strings.Join(answers, " ")
}

// ExtractAnswersByBlock извлекает ответы по блокам
func (i *Interview) ExtractAnswersByBlock() map[string]string {
	blockAnswers := make(map[string]string)

	for _, block := range i.Blocks {
		var answers []string
		for _, qa := range block.QuestionsAndAnswers {
			if strings.TrimSpace(qa.Answer) != "" {
				answers = append(answers, qa.Answer)
			}
		}
		blockAnswers[block.BlockName] = strings.Join(answers, " ")
	}

	return blockAnswers
}

// ExtractContextualAnswers извлекает ответы с контекстом вопросов
func (i *Interview) ExtractContextualAnswers() string {
	var contextualText []string

	for _, block := range i.Blocks {
		// Добавляем название блока как контекст
		blockTitle := formatBlockName(block.BlockName)
		contextualText = append(contextualText, fmt.Sprintf("=== %s ===", blockTitle))

		for _, qa := range block.QuestionsAndAnswers {
			if strings.TrimSpace(qa.Answer) != "" {
				// Добавляем вопрос как контекст для лучшего понимания
				contextualText = append(contextualText, fmt.Sprintf("На вопрос: %s", qa.Question))
				contextualText = append(contextualText, fmt.Sprintf("Ответ: %s", qa.Answer))
				contextualText = append(contextualText, "") // Пустая строка для разделения
			}
		}
	}

	return strings.Join(contextualText, "\n")
}

// formatBlockName преобразует техническое название блока в читаемое
func formatBlockName(blockName string) string {
	blockNames := map[string]string{
		"childhood_family":  "Детство и семья",
		"education_career":  "Образование и карьера",
		"values_future":     "Ценности и планы на будущее",
		"relationships":     "Отношения",
		"achievements":      "Достижения",
		"challenges":        "Трудности и преодоление",
		"personality":       "Личностные особенности",
		"hobbies_interests": "Хобби и интересы",
	}

	if readable, exists := blockNames[blockName]; exists {
		return readable
	}

	// Если название не найдено, форматируем его
	formatted := strings.ReplaceAll(blockName, "_", " ")
	return strings.Title(formatted)
}

// GetInterviewMetadata возвращает метаданные интервью
func (i *Interview) GetInterviewMetadata() map[string]interface{} {
	totalQuestions := 0
	totalAnswers := 0

	for _, block := range i.Blocks {
		totalQuestions += len(block.QuestionsAndAnswers)
		for _, qa := range block.QuestionsAndAnswers {
			if strings.TrimSpace(qa.Answer) != "" {
				totalAnswers++
			}
		}
	}

	return map[string]interface{}{
		"interview_id":    i.InterviewID,
		"timestamp":       i.Timestamp,
		"total_blocks":    len(i.Blocks),
		"total_questions": totalQuestions,
		"total_answers":   totalAnswers,
		"completion_rate": float64(totalAnswers) / float64(totalQuestions) * 100,
	}
}

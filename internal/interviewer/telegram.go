package interviewer

import (
	"fmt"
	"interview-bot-complete/internal/config"
	"interview-bot-complete/internal/storage"
	"strings"
)

// GenerateQuestion генерирует следующий вопрос для текущего блока
func (s *Service) GenerateQuestion(block config.Block, currentDialogue []storage.QA, previousSummaries []string, cfg *config.Config) (string, error) {
	// Строим промпт для генерации вопроса
	prompt := s.buildQuestionPrompt(block, currentDialogue, previousSummaries, cfg)

	messages := []Message{
		{Role: "system", Content: prompt},
	}

	question, err := s.callOpenAI(messages, cfg)
	if err != nil {
		return "", fmt.Errorf("ошибка генерации вопроса: %w", err)
	}

	return strings.TrimSpace(question), nil
}

// CreateSummary создает саммари блока (используется из telegram handler)
func (s *Service) CreateSummary(dialogue []storage.QA, cfg *config.Config) (string, error) {
	return s.createSummary(dialogue, cfg)
}

// buildQuestionPrompt создает промпт для генерации одного вопроса
func (s *Service) buildQuestionPrompt(block config.Block, currentDialogue []storage.QA, previousSummaries []string, cfg *config.Config) string {
	var prompt strings.Builder

	// Базовая роль
	prompt.WriteString("Ты опытный психолог-интервьюер с 15-летним стажем, работающий через Telegram бот.\n\n")

	// Контекст блока
	prompt.WriteString(fmt.Sprintf("ТЕКУЩИЙ БЛОК: \"%s\" (%d/%d)\n", block.Title, block.ID, cfg.GetTotalBlocks()))
	prompt.WriteString(fmt.Sprintf("СТРАТЕГИЯ: %s\n\n", block.ContextPrompt))

	// Контекст из предыдущих блоков
	if len(previousSummaries) > 0 {
		prompt.WriteString("КОНТЕКСТ ИЗ ПРЕДЫДУЩИХ БЛОКОВ:\n")
		for i, summary := range previousSummaries {
			prompt.WriteString(fmt.Sprintf("Блок %d: %s\n", i+1, summary))
		}
		prompt.WriteString("\n")
	}

	// Текущий диалог в блоке
	if len(currentDialogue) > 0 {
		prompt.WriteString("ТЕКУЩИЙ ДИАЛОГ В БЛОКЕ:\n")
		for i, qa := range currentDialogue {
			prompt.WriteString(fmt.Sprintf("Вопрос %d: %s\n", i+1, qa.Question))
			if qa.Answer != "" {
				prompt.WriteString(fmt.Sprintf("Ответ %d: %s\n\n", i+1, qa.Answer))
			}
		}
	}

	// Инструкции
	maxQuestions := cfg.GetQuestionsPerBlock() + cfg.GetMaxFollowupQuestions()
	currentQuestionNum := len(currentDialogue) + 1

	prompt.WriteString("ТВОЯ ЗАДАЧА:\n")
	prompt.WriteString(fmt.Sprintf("- Ты на вопросе %d из максимум %d в этом блоке\n", currentQuestionNum, maxQuestions))

	if currentQuestionNum <= cfg.GetQuestionsPerBlock() {
		prompt.WriteString("- Это базовый вопрос - должен покрывать ключевые области блока\n")
	} else {
		prompt.WriteString("- Это уточняющий вопрос - углубись в интересные детали из предыдущих ответов\n")
	}

	prompt.WriteString("- Задай ОДИН конкретный вопрос\n")
	prompt.WriteString("- Вопрос должен быть понятным и располагающим к откровенности\n")
	prompt.WriteString("- Учитывай контекст предыдущих ответов\n")
	prompt.WriteString("- Создавай атмосферу доверия\n\n")

	// Области фокуса
	if len(block.FocusAreas) > 0 {
		prompt.WriteString("ОБЛАСТИ ФОКУСА ДЛЯ ЭТОГО БЛОКА:\n")
		for _, area := range block.FocusAreas {
			prompt.WriteString(fmt.Sprintf("- %s\n", area))
		}
		prompt.WriteString("\n")
	}

	// Условие завершения
	if currentQuestionNum >= maxQuestions {
		prompt.WriteString("ВНИМАНИЕ: Это последний вопрос в блоке. После него нужно будет переходить к следующему блоку.\n\n")
	}

	prompt.WriteString("ОТВЕТ: Напиши только текст вопроса, без дополнительных комментариев.")

	return prompt.String()
}

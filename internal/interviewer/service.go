package interviewer

import (
	"bufio"
	"fmt"
	"interview-bot-complete/internal/config"
	"interview-bot-complete/internal/storage"
	"net/http"
	"os"
	"strings"
)

// Service представляет сервис интервьюера
type Service struct {
	apiKey string
	client *http.Client
}

// New создает новый сервис интервьюера
func New(apiKey string) *Service {
	return &Service{
		apiKey: apiKey,
		client: &http.Client{},
	}
}

// ConductBlock проводит интервью для одного блока
func (s *Service) ConductBlock(block config.Block, previousSummaries []string, cfg *config.Config) (*storage.BlockResult, string, error) {
	// Подготавливаем промпт для интервьюера
	interviewPrompt := s.buildInterviewPrompt(block, previousSummaries, cfg)

	// Проводим интервью
	dialogue, err := s.conductInterview(interviewPrompt, cfg)
	if err != nil {
		return nil, "", fmt.Errorf("ошибка проведения интервью: %w", err)
	}

	// Создаем саммари блока
	summary, err := s.createSummary(dialogue, cfg)
	if err != nil {
		return nil, "", fmt.Errorf("ошибка создания саммари: %w", err)
	}

	// Создаем JSON результат блока
	blockResult, err := s.createBlockJSON(block, dialogue)
	if err != nil {
		return nil, "", fmt.Errorf("ошибка создания JSON блока: %w", err)
	}

	return blockResult, summary, nil
}

// buildInterviewPrompt создает промпт для интервьюера с учетом контекста
func (s *Service) buildInterviewPrompt(block config.Block, previousSummaries []string, cfg *config.Config) string {
	var prompt strings.Builder

	// Базовый промпт
	prompt.WriteString("Ты опытный психолог-интервьюер с 15-летним стажем. Твоя задача - максимально эффективно собрать информацию о человеке.\n\n")

	// Ограничения
	prompt.WriteString("ЖЕСТКИЕ ОГРАНИЧЕНИЯ:\n")
	prompt.WriteString(fmt.Sprintf("- У тебя МАКСИМУМ %d вопросов на этот блок (%d базовых + %d дополнительных)\n",
		cfg.GetQuestionsPerBlock()+cfg.GetMaxFollowupQuestions(),
		cfg.GetQuestionsPerBlock(),
		cfg.GetMaxFollowupQuestions()))
	prompt.WriteString("- Каждый вопрос должен извлекать максимум полезной информации\n")
	prompt.WriteString("- Используй техники глубинного интервью\n\n")

	// Информация о блоке
	prompt.WriteString(fmt.Sprintf("ТЕКУЩИЙ БЛОК: \"%s\" (%d/%d)\n\n", block.Title, block.ID, cfg.GetTotalBlocks()))

	// Контекст из предыдущих блоков
	if len(previousSummaries) > 0 {
		prompt.WriteString("КОНТЕКСТ ИЗ ПРЕДЫДУЩИХ БЛОКОВ:\n")
		for i, summary := range previousSummaries {
			prompt.WriteString(fmt.Sprintf("Блок %d: %s\n", i+1, summary))
		}
		prompt.WriteString("\n")
	}

	// Специфичный промпт блока
	prompt.WriteString("ТВОЯ СТРАТЕГИЯ:\n")
	prompt.WriteString(block.ContextPrompt)
	prompt.WriteString("\n\n")

	// Области фокуса
	if len(block.FocusAreas) > 0 {
		prompt.WriteString("ОБЯЗАТЕЛЬНО ПОКРОЙ:\n")
		for _, area := range block.FocusAreas {
			prompt.WriteString(fmt.Sprintf("- %s\n", area))
		}
		prompt.WriteString("\n")
	}

	// Стиль и финальные инструкции
	prompt.WriteString("СТИЛЬ: Профессиональный, но теплый. Создавай атмосферу доверия.\n\n")
	prompt.WriteString("ПЕРЕХОД К СЛЕДУЮЩЕМУ БЛОКУ:\n")
	prompt.WriteString(fmt.Sprintf("После завершения %d вопросов сделай плавный переход к следующей теме, не упоминая номера блоков.\n\n",
		cfg.GetQuestionsPerBlock()+cfg.GetMaxFollowupQuestions()))
	prompt.WriteString(fmt.Sprintf("ПОСЛЕ %d ВОПРОСОВ - ЗАВЕРШАЙ БЛОК. НЕ ПРЕВЫШАЙ ЛИМИТ.\n\n",
		cfg.GetQuestionsPerBlock()+cfg.GetMaxFollowupQuestions()))
	prompt.WriteString("Начинай с первого вопроса. Задавай по одному вопросу за раз.")

	return prompt.String()
}

// conductInterview проводит диалог с пользователем
func (s *Service) conductInterview(systemPrompt string, cfg *config.Config) ([]storage.QA, error) {
	var dialogue []storage.QA
	scanner := bufio.NewScanner(os.Stdin)

	// Инициализируем диалог с системным промптом
	messages := []Message{
		{Role: "system", Content: systemPrompt},
	}

	questionCount := 0
	maxQuestions := cfg.GetQuestionsPerBlock() + cfg.GetMaxFollowupQuestions()

	for questionCount < maxQuestions {
		// Получаем вопрос от AI
		response, err := s.callOpenAI(messages, cfg)
		if err != nil {
			return nil, fmt.Errorf("ошибка вызова OpenAI: %w", err)
		}

		question := strings.TrimSpace(response)

		// Проверяем, не завершает ли AI блок
		if strings.Contains(strings.ToLower(question), "завершаем") ||
			strings.Contains(strings.ToLower(question), "переходим") ||
			strings.Contains(strings.ToLower(question), "блок завершен") {
			break
		}

		// Выводим вопрос
		fmt.Printf("Вопрос: %s\n", question)
		fmt.Print("Ваш ответ: ")

		// Читаем ответ пользователя
		if !scanner.Scan() {
			break
		}
		answer := strings.TrimSpace(scanner.Text())

		if answer == "" {
			fmt.Println("Пожалуйста, дайте ответ.")
			continue
		}

		// Сохраняем вопрос и ответ
		dialogue = append(dialogue, storage.QA{
			Question: question,
			Answer:   answer,
		})

		// Добавляем в контекст для следующего вопроса
		messages = append(messages,
			Message{Role: "assistant", Content: question},
			Message{Role: "user", Content: answer},
		)

		questionCount++
	}

	return dialogue, nil
}

// createSummary создает саммари блока
func (s *Service) createSummary(dialogue []storage.QA, cfg *config.Config) (string, error) {
	prompt := s.buildSummaryPrompt(dialogue)

	messages := []Message{
		{Role: "system", Content: prompt},
	}

	summary, err := s.callOpenAI(messages, cfg)
	if err != nil {
		return "", fmt.Errorf("ошибка создания саммари: %w", err)
	}

	return summary, nil
}

// buildSummaryPrompt создает промпт для саммаризации
func (s *Service) buildSummaryPrompt(dialogue []storage.QA) string {
	var prompt strings.Builder

	prompt.WriteString("Ты опытный психолог-аналитик. Проанализируй прошедший блок интервью и создай структурированное саммари.\n\n")

	prompt.WriteString("ВОПРОСЫ И ОТВЕТЫ:\n")
	for i, qa := range dialogue {
		prompt.WriteString(fmt.Sprintf("%d. Вопрос: %s\n", i+1, qa.Question))
		prompt.WriteString(fmt.Sprintf("   Ответ: %s\n\n", qa.Answer))
	}

	prompt.WriteString("ЗАДАЧА: Извлечь максимум полезной информации для следующих блоков интервью.\n\n")

	prompt.WriteString("СОЗДАЙ КРАТКОЕ САММАРИ В СВОБОДНОЙ ФОРМЕ:\n")
	prompt.WriteString("- Ключевые факты о человеке\n")
	prompt.WriteString("- Важные темы и приоритеты\n")
	prompt.WriteString("- Эмоциональные реакции и чувствительные области\n")
	prompt.WriteString("- Паттерны поведения и ценности\n\n")
	prompt.WriteString("ВАЖНО: Будь конкретным, избегай общих фраз. Информация должна быть полезна для адаптации следующих блоков.")

	return prompt.String()
}

// createBlockJSON создает JSON результат блока
func (s *Service) createBlockJSON(block config.Block, dialogue []storage.QA) (*storage.BlockResult, error) {
	return &storage.BlockResult{
		BlockID:             block.ID,
		BlockName:           block.Name,
		QuestionsAndAnswers: dialogue,
	}, nil
}

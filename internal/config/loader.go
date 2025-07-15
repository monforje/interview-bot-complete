package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Load загружает конфигурацию из YAML файла
func Load(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения файла %s: %w", filename, err)
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("ошибка парсинга YAML: %w", err)
	}

	// Валидация конфигурации
	err = validateConfig(&config)
	if err != nil {
		return nil, fmt.Errorf("ошибка валидации конфигурации: %w", err)
	}

	return &config, nil
}

// validateConfig проверяет корректность конфигурации
func validateConfig(config *Config) error {
	if config.InterviewConfig.TotalBlocks <= 0 {
		return fmt.Errorf("total_blocks должно быть больше 0")
	}

	if config.InterviewConfig.QuestionsPerBlock <= 0 {
		return fmt.Errorf("questions_per_block должно быть больше 0")
	}

	if config.InterviewConfig.MaxFollowupQuestions < 0 {
		return fmt.Errorf("max_followup_questions не может быть отрицательным")
	}

	if len(config.Blocks) != config.InterviewConfig.TotalBlocks {
		return fmt.Errorf("количество блоков (%d) не соответствует total_blocks (%d)",
			len(config.Blocks), config.InterviewConfig.TotalBlocks)
	}

	if len(config.ProfileFields) == 0 {
		return fmt.Errorf("profile_fields не должны быть пустыми")
	}
	if len(config.ProfileFields) > 100 {
		return fmt.Errorf("profile_fields не должны превышать 100 элементов")
	}

	// Проверяем ID блоков и вопросы
	for i, block := range config.Blocks {
		expectedID := i + 1
		if block.ID != expectedID {
			return fmt.Errorf("блок %d имеет неверный ID: ожидался %d, получен %d",
				i, expectedID, block.ID)
		}

		if block.Name == "" {
			return fmt.Errorf("блок %d должен иметь name", block.ID)
		}

		if block.Title == "" {
			return fmt.Errorf("блок %d должен иметь title", block.ID)
		}

		if block.ContextPrompt == "" {
			return fmt.Errorf("блок %d должен иметь context_prompt", block.ID)
		}

		if len(block.Questions) != config.InterviewConfig.QuestionsPerBlock {
			return fmt.Errorf("блок %d должен содержать %d вопросов, найдено %d", block.ID, config.InterviewConfig.QuestionsPerBlock, len(block.Questions))
		}
	}

	return nil
}

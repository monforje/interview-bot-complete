package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const resultsDir = "results"

// SaveResult сохраняет результат интервью в JSON файл
func SaveResult(result *InterviewResult) error {
	// Создаем директорию если её нет
	err := os.MkdirAll(resultsDir, 0755)
	if err != nil {
		return fmt.Errorf("ошибка создания директории %s: %w", resultsDir, err)
	}

	// Формируем имя файла
	filename := fmt.Sprintf("interview_%s.json", result.InterviewID)
	filepath := filepath.Join(resultsDir, filename)

	// Сериализуем результат в JSON с отступами
	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("ошибка сериализации результата: %w", err)
	}

	// Записываем в файл
	err = os.WriteFile(filepath, jsonData, 0644)
	if err != nil {
		return fmt.Errorf("ошибка записи файла %s: %w", filepath, err)
	}

	return nil
}

// LoadResult загружает результат интервью из JSON файла
func LoadResult(interviewID string) (*InterviewResult, error) {
	filename := fmt.Sprintf("interview_%s.json", interviewID)
	filepath := filepath.Join(resultsDir, filename)

	// Читаем файл
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения файла %s: %w", filepath, err)
	}

	// Десериализуем JSON
	var result InterviewResult
	err = json.Unmarshal(data, &result)
	if err != nil {
		return nil, fmt.Errorf("ошибка десериализации JSON: %w", err)
	}

	return &result, nil
}

// ListResults возвращает список всех сохраненных интервью
func ListResults() ([]string, error) {
	// Проверяем существование директории
	if _, err := os.Stat(resultsDir); os.IsNotExist(err) {
		return []string{}, nil
	}

	// Читаем содержимое директории
	entries, err := os.ReadDir(resultsDir)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения директории %s: %w", resultsDir, err)
	}

	var results []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			// Извлекаем ID интервью из имени файла
			name := entry.Name()
			if len(name) > 10 && name[:10] == "interview_" {
				interviewID := name[10 : len(name)-5] // убираем "interview_" и ".json"
				results = append(results, interviewID)
			}
		}
	}

	return results, nil
}

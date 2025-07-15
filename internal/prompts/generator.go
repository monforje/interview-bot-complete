package prompts

import (
	"fmt"
	"strings"

	"interview-bot-complete/internal/schema"
)

// GenerateOptimizedExtractionPrompt - оптимизированный промпт для извлечения профиля за один запрос
func GenerateOptimizedExtractionPrompt(schemaFields map[string]schema.SchemaField, userText string) string {
	prompt := `Создай профиль пользователя в формате JSON на основе текста интервью.

ИНСТРУКЦИИ:
1. Заполни все поля из списка ниже
2. Если информации нет в тексте - ставь null
3. Массивы должны содержать конкретные значения, не общие фразы
4. Числовые поля должны быть числами, строковые - строками
5. Будь точным и конкретным
6. Верни ТОЛЬКО валидный JSON, без markdown и комментариев

ПОЛЯ ДЛЯ ЗАПОЛНЕНИЯ:
%s

ПРАВИЛА ЗАПОЛНЕНИЯ:
- name: полное имя пользователя
- age: возраст числом
- birth_city/current_city: названия городов
- hard_skills: конкретные технические навыки ["Python", "React", "SQL"]
- soft_skills: личностные качества ["коммуникабельность", "лидерство"] 
- hobbies: конкретные хобби ["футбол", "фотография", "программирование"]
- personality_traits: черты характера ["целеустремленный", "творческий"]
- values: жизненные ценности ["семья", "развитие", "честность"]
- career_goals: карьерные цели ["стать тимлидом", "открыть стартап"]

ТЕКСТ ИНТЕРВЬЮ:
%s

ОТВЕТ (только JSON):`

	schemaDescription := generateSchemaDescription(schemaFields)
	return fmt.Sprintf(prompt, schemaDescription, userText)
}

func generateSchemaDescription(schemaFields map[string]schema.SchemaField) string {
	var builder strings.Builder

	// Сортируем поля по важности для лучшего восприятия
	importantFields := []string{"name", "age", "birth_city", "current_city", "university", "current_position"}

	// Сначала важные поля
	for _, fieldName := range importantFields {
		if field, exists := schemaFields[fieldName]; exists {
			appendFieldDescription(&builder, field)
		}
	}

	// Затем остальные поля
	for _, field := range schemaFields {
		isImportant := false
		for _, importantField := range importantFields {
			if field.Name == importantField {
				isImportant = true
				break
			}
		}
		if !isImportant {
			appendFieldDescription(&builder, field)
		}
	}

	return builder.String()
}

func appendFieldDescription(builder *strings.Builder, field schema.SchemaField) {
	if field.IsArray {
		builder.WriteString(fmt.Sprintf("- %s: [] (массив)\n", field.Name))
	} else if field.IsObject {
		builder.WriteString(fmt.Sprintf("- %s: {} (объект)\n", field.Name))
	} else {
		builder.WriteString(fmt.Sprintf("- %s: %s\n", field.Name, field.Type))
	}
}

// Удаляем старые неиспользуемые функции
// GenerateValidationPrompt больше не нужен - валидация происходит локально
// GenerateProfileMatchPrompt больше не нужен - убираем типы личности

package prompts

import (
	"fmt"
	"strings"

	"interview-bot-complete/internal/schema"
)

func GenerateExtractionPrompt(schemaFields map[string]schema.SchemaField, userText string) string {
	prompt := `Заполни все поля из списка profile_fields на основе текста пользователя. Если данных нет — ставь null. Верни только валидный JSON-объект с этими полями, без markdown и комментариев.

СПИСОК ПОЛЕЙ:
%s

ТЕКСТ ПОЛЬЗОВАТЕЛЯ:
%s

ОТВЕТ (только JSON):`

	schemaDescription := generateSchemaDescription(schemaFields)
	return fmt.Sprintf(prompt, schemaDescription, userText)
}

func GenerateValidationPrompt(profileJSON string) string {
	return fmt.Sprintf(`Ты эксперт по валидации данных. Проверь профиль и исправь найденные проблемы.

ПРОВЕРКИ:
1. ДУБЛИРОВАНИЕ: Удали из "tags" информацию, которая дублируется с основными полями
2. ТИПЫ ДАННЫХ: Убедись, что все поля соответствуют нужным типам
3. ЛОГИКА: Проверь на противоречия (например, age: 25 и education.year: 2030)
4. СТРУКТУРА: Убедись, что JSON валиден и правильно структурирован
5. КОНСИСТЕНТНОСТЬ: Проверь логическую связность данных

ПРАВИЛА ИСПРАВЛЕНИЯ:
- Приоритет у основных полей, теги - вторичны
- Удаляй дубли из тегов, не перемещай информацию
- Исправляй типы данных без потери смысла
- Сохраняй только логически корректную информацию
- Если поле должно быть числом, но пришла строка - попробуй преобразовать

ПРИМЕРЫ ПРОБЛЕМ И РЕШЕНИЙ:
- Дубль: skills: [{"name": "Go"}] + tags: {"programming": "Go"} → удали тег
- Тип: age: "25" → age: 25
- Противоречие: age: 20, experience_years: 10 → исправь experience_years: 2

ПРОФИЛЬ ДЛЯ ПРОВЕРКИ:
%s

ОТВЕТ (чистый исправленный JSON без markdown оформления, без markdown блоков и трех обратных кавычек):`, profileJSON)
}

func generateSchemaDescription(schemaFields map[string]schema.SchemaField) string {
	var builder strings.Builder

	for _, field := range schemaFields {
		if field.IsArray {
			builder.WriteString(fmt.Sprintf("- %s: array\n", field.Name))
		} else if field.IsObject {
			builder.WriteString(fmt.Sprintf("- %s: object\n", field.Name))
		} else {
			builder.WriteString(fmt.Sprintf("- %s: %s\n", field.Name, field.Type))
		}
	}

	return builder.String()
}

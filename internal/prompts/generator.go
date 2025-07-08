package prompts

import (
	"fmt"
	"strings"

	"interview-bot-complete/internal/schema"
)

func GenerateExtractionPrompt(schemaFields map[string]schema.SchemaField, userText string) string {
	prompt := `Ты профессиональный экстрактор данных. Проанализируй текст пользователя и заполни профиль в формате JSON.

СХЕМА ДАННЫХ:
%s

ПРАВИЛА ЗАПОЛНЕНИЯ:

1. ФИКСИРОВАННЫЕ ПОЛЯ: Извлеки только те поля, которые есть в схеме выше
2. ТИПЫ ДАННЫХ: Строго соблюдай указанные типы (string, int, array, object)
3. ТОЧЕЧНАЯ НОТАЦИЯ: Поля вида "location.city" создавай как вложенные объекты {"location": {"city": "значение"}}
4. МАССИВЫ: Поля типа array создавай как массивы объектов
5. ОБЯЗАТЕЛЬНЫЕ ПОЛЯ: Если данных нет - ставь null, НЕ ПРИДУМЫВАЙ
6. ТЕГИ: После заполнения основных полей создай section "tags" для дополнительной информации

ПРИМЕРЫ ПРАВИЛЬНЫХ СТРУКТУР:
- education: array → "education": [{"university": "МГУ", "degree": "бакалавр", "year": 2020}]
- skills: array → "skills": [{"name": "Go", "level": "advanced"}, {"name": "Python", "level": "intermediate"}]
- location.city: string → "location": {"city": "Москва"}
- social.telegram: string → "social": {"telegram": "@username"}
- tags: object → "tags": {"hobby": "фотография", "personality": "коммуникабельный"}

ВАЖНО:
- Возвращай ТОЛЬКО валидный JSON без markdown блоков и трех обратных кавычек
- Никаких дополнительных комментариев или объяснений
- Для полей типа array создавай объекты с осмысленными ключами
- Теги используй для информации, которая не поместилась в стандартные поля
- Не дублируй информацию между основными полями и тегами

ТЕКСТ ПОЛЬЗОВАТЕЛЯ:
%s

ОТВЕТ (чистый JSON без оформления, без markdown блоков и трех обратных кавычек):`

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

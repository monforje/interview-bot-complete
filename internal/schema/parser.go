package schema

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v2"
)

type SchemaField struct {
	Name     string
	Type     string
	IsArray  bool
	IsObject bool
	Nested   map[string]SchemaField
}

func ParseYAMLSchema(yamlContent []byte) (map[string]SchemaField, error) {
	schema := make(map[string]interface{})
	err := yaml.Unmarshal(yamlContent, &schema)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling YAML: %w", err)
	}

	result := make(map[string]SchemaField)

	for key, value := range schema {
		field := SchemaField{
			Name: key,
			Type: parseType(value),
		}

		// Обработка точечной нотации
		if strings.Contains(key, ".") {
			field = parseNestedField(key, value)
		}

		// Определение массивов и объектов
		if field.Type == "array" {
			field.IsArray = true
		}
		if field.Type == "object" {
			field.IsObject = true
		}

		result[key] = field
	}

	return result, nil
}

func parseType(value interface{}) string {
	switch v := value.(type) {
	case string:
		return v
	case int:
		return "int"
	case float64:
		return "float"
	case bool:
		return "bool"
	case []interface{}:
		return "array"
	case map[string]interface{}:
		return "object"
	default:
		return "string" // default fallback
	}
}

func parseNestedField(dotKey string, value interface{}) SchemaField {
	// Для точечной нотации создаем структуру вложенных полей
	// location.city → {"location": {"city": "string"}}

	// Возвращаем только корневое поле, остальная логика в промптах
	return SchemaField{
		Name:     dotKey, // Сохраняем исходное имя для обработки в промптах
		Type:     parseType(value),
		IsObject: false,
		Nested:   nil,
	}
}

func (s SchemaField) String() string {
	if s.IsArray {
		return fmt.Sprintf("%s: array", s.Name)
	}
	if s.IsObject && len(s.Nested) > 0 {
		return fmt.Sprintf("%s: object with nested fields", s.Name)
	}
	return fmt.Sprintf("%s: %s", s.Name, s.Type)
}

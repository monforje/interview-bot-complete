package validator

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"interview-bot-complete/internal/schema"
)

func ValidateProfileJSON(jsonStr string, schemaFields map[string]schema.SchemaField) error {
	// Проверка валидности JSON
	var profile map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &profile); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	// Проверка типов данных
	for key, field := range schemaFields {
		if value, exists := profile[key]; exists && value != nil {
			if err := validateFieldType(value, field, key); err != nil {
				return fmt.Errorf("field %s: %w", key, err)
			}
		}
	}

	// Проверка вложенных объектов для точечной нотации
	if err := validateNestedFields(profile, schemaFields); err != nil {
		return fmt.Errorf("nested fields validation: %w", err)
	}

	return nil
}

func validateFieldType(value interface{}, field schema.SchemaField, fieldName string) error {
	// Обработка точечной нотации - НЕ рекурсивно!
	if strings.Contains(fieldName, ".") {
		// Для точечной нотации просто проверяем базовый тип значения
		return validateBasicType(value, field.Type)
	}

	return validateBasicType(value, field.Type)
}

func validateBasicType(value interface{}, fieldType string) error {
	switch fieldType {
	case "string":
		if _, ok := value.(string); !ok {
			return fmt.Errorf("expected string, got %T", value)
		}
	case "int":
		// JSON unmarshals numbers as float64
		if v, ok := value.(float64); ok {
			// Проверяем, что это целое число
			if v != float64(int(v)) {
				return fmt.Errorf("expected integer, got float %f", v)
			}
		} else {
			return fmt.Errorf("expected number, got %T", value)
		}
	case "float":
		if _, ok := value.(float64); !ok {
			return fmt.Errorf("expected number, got %T", value)
		}
	case "bool":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("expected boolean, got %T", value)
		}
	case "array":
		if _, ok := value.([]interface{}); !ok {
			return fmt.Errorf("expected array, got %T", value)
		}
	case "object":
		if _, ok := value.(map[string]interface{}); !ok {
			return fmt.Errorf("expected object, got %T", value)
		}
	}

	return nil
}

func validateDotNotationField(value interface{}, field schema.SchemaField, fieldName string) error {
	// Убираем рекурсию - просто валидируем базовый тип
	return validateBasicType(value, field.Type)
}

func validateNestedFields(profile map[string]interface{}, schemaFields map[string]schema.SchemaField) error {
	for key, field := range schemaFields {
		if !strings.Contains(key, ".") {
			continue
		}

		// Проверяем, что поля с точечной нотацией действительно создали вложенные объекты
		parts := strings.Split(key, ".")
		if len(parts) != 2 {
			continue // Поддерживаем только одноуровневую вложенность
		}

		parentKey := parts[0]
		childKey := parts[1]

		if parentValue, exists := profile[parentKey]; exists && parentValue != nil {
			parentObj, ok := parentValue.(map[string]interface{})
			if !ok {
				return fmt.Errorf("field %s should be an object for nested field %s", parentKey, key)
			}

			if childValue, childExists := parentObj[childKey]; childExists && childValue != nil {
				if err := validateFieldType(childValue, field, key); err != nil {
					return fmt.Errorf("nested field %s: %w", key, err)
				}
			}
		}
	}

	return nil
}

func PrettyPrintValidationResult(jsonStr string) {
	var profile map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &profile); err != nil {
		fmt.Printf("Error parsing JSON: %v\n", err)
		return
	}

	fmt.Println("Profile validation result:")
	printMap(profile, 0)
}

func printMap(m map[string]interface{}, indent int) {
	spaces := strings.Repeat("  ", indent)
	for key, value := range m {
		switch v := value.(type) {
		case map[string]interface{}:
			fmt.Printf("%s%s: (object)\n", spaces, key)
			printMap(v, indent+1)
		case []interface{}:
			fmt.Printf("%s%s: (array with %d items)\n", spaces, key, len(v))
			for i, item := range v {
				if itemMap, ok := item.(map[string]interface{}); ok {
					fmt.Printf("%s  [%d]:\n", spaces, i)
					printMap(itemMap, indent+2)
				} else {
					fmt.Printf("%s  [%d]: %v (%s)\n", spaces, i, item, reflect.TypeOf(item))
				}
			}
		case nil:
			fmt.Printf("%s%s: null\n", spaces, key)
		default:
			fmt.Printf("%s%s: %v (%s)\n", spaces, key, value, reflect.TypeOf(value))
		}
	}
}

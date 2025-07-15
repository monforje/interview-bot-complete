package prompts

import "fmt"

func GenerateProfileMatchPrompt(profileJSON string) string {
	return fmt.Sprintf(`
Ты — аналитик психологического профиля. На входе у тебя JSON, содержащий описание личности, ценностей, поведения и мотиваций пользователя.

На основе этих данных, определи, какой архетип или тип личности наиболее соответствует этому профилю.

Ответь строго в формате JSON:
{
  "profile_match.name": "Название архетипа",
  "profile_match.id": "Идентификатор (на англ.)",
  "profile_match.summary": "Краткое объяснение, почему выбран этот тип"
}

Вот входной профиль:
%s
`, profileJSON)
}

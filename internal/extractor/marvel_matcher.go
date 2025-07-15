package extractor

import (
	"encoding/json"
	"fmt"
)

// ProfileMatch — структура подмножества полей из результата анализа
type ProfileMatch struct {
	MatchID string `json:"profile_match.id"`
	Name    string `json:"profile_match.name"`
	Summary string `json:"profile_match.summary"`
}

// ParseProfileMatch — извлечение финального профиля из json
func ParseProfileMatch(jsonData []byte) (*ProfileMatch, error) {
	var profile ProfileMatch
	if err := json.Unmarshal(jsonData, &profile); err != nil {
		return nil, err
	}
	return &profile, nil
}

// GenerateProfileDescription — строка для вывода пользователю
func GenerateProfileDescription(profile *ProfileMatch) string {
	return fmt.Sprintf(
		"🧑‍💼 *Ваш тип личности — %s* (%s)\n\n%s",
		profile.Name,
		profile.MatchID,
		profile.Summary,
	)
}

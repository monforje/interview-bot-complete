package config

// Config представляет конфигурацию интервью
type Config struct {
	InterviewConfig  InterviewConfig  `yaml:"interview_config"`
	Blocks           []Block          `yaml:"blocks"`
	ProfileFields    []string         `yaml:"profile_fields"`
	SummaryStructure SummaryStructure `yaml:"summary_structure"`
}

// InterviewConfig содержит общие настройки интервью
type InterviewConfig struct {
	TotalBlocks          int `yaml:"total_blocks"`
	QuestionsPerBlock    int `yaml:"questions_per_block"`
	MaxFollowupQuestions int `yaml:"max_followup_questions"`
}

// Block представляет один блок интервью
type Block struct {
	ID            int      `yaml:"id"`
	Name          string   `yaml:"name"`
	Title         string   `yaml:"title"`
	ContextPrompt string   `yaml:"context_prompt"`
	FocusAreas    []string `yaml:"focus_areas"`
	Questions     []string `yaml:"questions"`
}

// SummaryStructure определяет структуру саммари
type SummaryStructure struct {
	KeyFacts           []string `yaml:"key_facts"`
	ImportantThemes    []string `yaml:"important_themes"`
	EmotionalMarkers   []string `yaml:"emotional_markers"`
	BehavioralPatterns []string `yaml:"behavioral_patterns"`
	ValuesBeliefs      []string `yaml:"values_beliefs"`
	Priorities         []string `yaml:"priorities"`
	SensitiveTopics    []string `yaml:"sensitive_topics"`
}

// Методы для удобного доступа к конфигурации
func (c *Config) GetTotalBlocks() int {
	return c.InterviewConfig.TotalBlocks
}

func (c *Config) GetQuestionsPerBlock() int {
	return c.InterviewConfig.QuestionsPerBlock
}

func (c *Config) GetMaxFollowupQuestions() int {
	return c.InterviewConfig.MaxFollowupQuestions
}

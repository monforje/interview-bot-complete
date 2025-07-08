package storage

// InterviewResult представляет результат всего интервью
type InterviewResult struct {
	InterviewID string        `json:"interview_id"`
	Timestamp   string        `json:"timestamp"`
	Blocks      []BlockResult `json:"blocks"`
}

// BlockResult представляет результат одного блока
type BlockResult struct {
	BlockID             int    `json:"block_id"`
	BlockName           string `json:"block_name"`
	QuestionsAndAnswers []QA   `json:"questions_and_answers"`
}

// QA представляет один вопрос и ответ
type QA struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

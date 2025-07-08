package telegram

import (
	"interview-bot-complete/internal/storage"
	"time"
)

// Bot представляет Telegram бота
type Bot struct {
	token   string
	baseURL string
}

// Update представляет обновление от Telegram
type Update struct {
	UpdateID int      `json:"update_id"`
	Message  *Message `json:"message,omitempty"`
}

// Message представляет сообщение в Telegram
type Message struct {
	MessageID int    `json:"message_id"`
	From      *User  `json:"from,omitempty"`
	Chat      *Chat  `json:"chat"`
	Text      string `json:"text,omitempty"`
}

// User представляет пользователя Telegram
type User struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name,omitempty"`
	Username  string `json:"username,omitempty"`
}

// Chat представляет чат в Telegram
type Chat struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
	Username  string `json:"username,omitempty"`
	Type      string `json:"type"`
}

// SendMessageRequest представляет запрос на отправку сообщения
type SendMessageRequest struct {
	ChatID    int64  `json:"chat_id"`
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode,omitempty"`
}

// GetUpdatesResponse представляет ответ от getUpdates
type GetUpdatesResponse struct {
	OK     bool     `json:"ok"`
	Result []Update `json:"result"`
}

// SendMessageResponse представляет ответ от sendMessage
type SendMessageResponse struct {
	OK     bool     `json:"ok"`
	Result *Message `json:"result,omitempty"`
}

// Обновить UserSession
type UserSession struct {
	UserID              int64                    `json:"user_id"`
	InterviewID         string                   `json:"interview_id"`
	CurrentBlock        int                      `json:"current_block"`
	QuestionCount       int                      `json:"question_count"`
	State               SessionState             `json:"state"`
	CurrentDialogue     []storage.QA             `json:"current_dialogue"`
	CumulativeSummaries []string                 `json:"cumulative_summaries"`
	Result              *storage.InterviewResult `json:"result"`
	LastActivity        time.Time                `json:"last_activity"`
}

// SessionState представляет состояние сессии
type SessionState string

const (
	StateIdle          SessionState = "idle"
	StateInterview     SessionState = "interview"
	StateWaitingAnswer SessionState = "waiting_answer"
	StateCompleted     SessionState = "completed"
)

package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// New создает новый Telegram бот
func New(token string) *Bot {
	return &Bot{
		token:   token,
		baseURL: fmt.Sprintf("https://api.telegram.org/bot%s", token),
	}
}

// GetUpdates получает обновления от Telegram
func (b *Bot) GetUpdates(offset int) ([]Update, error) {
	url := fmt.Sprintf("%s/getUpdates?offset=%d&timeout=30", b.baseURL, offset)

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса getUpdates: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения ответа: %w", err)
	}

	var response GetUpdatesResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, fmt.Errorf("ошибка парсинга JSON: %w", err)
	}

	if !response.OK {
		return nil, fmt.Errorf("Telegram API вернул ошибку")
	}

	return response.Result, nil
}

// SendMessage отправляет сообщение пользователю
func (b *Bot) SendMessage(chatID int64, text string) error {
	request := SendMessageRequest{
		ChatID:    chatID,
		Text:      text,
		ParseMode: "Markdown",
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("ошибка сериализации запроса: %w", err)
	}

	url := fmt.Sprintf("%s/sendMessage", b.baseURL)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("ошибка отправки сообщения: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("ошибка чтения ответа: %w", err)
	}

	var response SendMessageResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return fmt.Errorf("ошибка парсинга ответа: %w", err)
	}

	if !response.OK {
		return fmt.Errorf("Telegram API вернул ошибку при отправке сообщения")
	}

	return nil
}

// SendFormattedMessage отправляет форматированное сообщение
func (b *Bot) SendFormattedMessage(chatID int64, format string, args ...interface{}) error {
	text := fmt.Sprintf(format, args...)
	return b.SendMessage(chatID, text)
}

// StartPolling запускает polling для получения обновлений
func (b *Bot) StartPolling(handler func(Update)) error {
	offset := 0

	for {
		updates, err := b.GetUpdates(offset)
		if err != nil {
			fmt.Printf("Ошибка получения обновлений: %v\n", err)
			time.Sleep(5 * time.Second)
			continue
		}

		for _, update := range updates {
			offset = update.UpdateID + 1
			go handler(update)
		}

		if len(updates) == 0 {
			time.Sleep(1 * time.Second)
		}
	}
}

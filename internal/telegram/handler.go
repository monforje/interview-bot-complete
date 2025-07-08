package telegram

import (
	"fmt"
	"interview-bot-complete/internal/config"
	"interview-bot-complete/internal/extractor"
	"interview-bot-complete/internal/interviewer"
	"interview-bot-complete/internal/storage"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type RateLimiter struct {
	requests map[int64][]time.Time
	mutex    sync.RWMutex
	limit    int
	window   time.Duration
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		requests: make(map[int64][]time.Time),
		limit:    limit,
		window:   window,
	}
}

func (rl *RateLimiter) IsAllowed(userID int64) bool {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	now := time.Now()

	// Очищаем старые запросы
	if requests, exists := rl.requests[userID]; exists {
		var validRequests []time.Time
		for _, reqTime := range requests {
			if now.Sub(reqTime) < rl.window {
				validRequests = append(validRequests, reqTime)
			}
		}
		rl.requests[userID] = validRequests
	}

	// Проверяем лимит
	if len(rl.requests[userID]) >= rl.limit {
		return false
	}

	// Добавляем новый запрос
	rl.requests[userID] = append(rl.requests[userID], now)
	return true
}

// Добавить очистку неактивных сессий
func (h *Handler) startSessionCleanup() {
	ticker := time.NewTicker(1 * time.Hour)
	go func() {
		for range ticker.C {
			h.cleanupInactiveSessions()
		}
	}()
}

func (h *Handler) cleanupInactiveSessions() {
	h.sessionsMutex.Lock()
	defer h.sessionsMutex.Unlock()

	cutoff := time.Now().Add(-24 * time.Hour) // Удаляем сессии старше 24 часов

	for userID, session := range h.sessions {
		if session.LastActivity.Before(cutoff) {
			delete(h.sessions, userID)
		}
	}
}

// Handler обрабатывает сообщения от пользователей
type Handler struct {
	bot           *Bot
	config        *config.Config
	interviewer   *interviewer.Service
	extractor     *extractor.Service
	sessions      map[int64]*UserSession
	sessionsMutex sync.RWMutex
	rateLimiter   *RateLimiter
}

// NewHandler создает новый обработчик
func NewHandler(bot *Bot, cfg *config.Config, interviewerService *interviewer.Service, extractorService *extractor.Service) *Handler {
	return &Handler{
		bot:         bot,
		config:      cfg,
		interviewer: interviewerService,
		extractor:   extractorService,
		sessions:    make(map[int64]*UserSession),
		rateLimiter: NewRateLimiter(10, time.Minute), // 10 сообщений в минуту

	}
}

// HandleUpdate обрабатывает обновление от Telegram
func (h *Handler) HandleUpdate(update Update) {
	if update.Message == nil || update.Message.From == nil {
		return
	}

	userID := update.Message.From.ID
	chatID := update.Message.Chat.ID
	text := strings.TrimSpace(update.Message.Text)

	// Получаем или создаем сессию пользователя
	session := h.getOrCreateSession(userID)

	// Обрабатываем команды
	if strings.HasPrefix(text, "/") {
		h.handleCommand(chatID, text, session)
		return
	}

	// Обрабатываем ответы пользователя
	h.handleUserInput(chatID, text, session)
}

// handleCommand обрабатывает команды бота
func (h *Handler) handleCommand(chatID int64, command string, session *UserSession) {
	switch command {
	case "/start":
		h.handleStartCommand(chatID, session)
	case "/help":
		h.handleHelpCommand(chatID)
	case "/status":
		h.handleStatusCommand(chatID, session)
	case "/restart":
		h.handleRestartCommand(chatID, session)
	case "/stop":
		h.handleStopCommand(chatID, session)
	default:
		h.bot.SendMessage(chatID, "Неизвестная команда. Используйте /help для получения списка команд.")
	}
}

// handleStartCommand обрабатывает команду /start
func (h *Handler) handleStartCommand(chatID int64, session *UserSession) {
	if session.State == StateInterview || session.State == StateWaitingAnswer {
		h.bot.SendMessage(chatID, "У вас уже идет интервью. Используйте /status для проверки прогресса или /restart для начала нового интервью.")
		return
	}

	// Инициализируем новое интервью
	h.initializeInterview(chatID, session)
}

// handleHelpCommand обрабатывает команду /help
func (h *Handler) handleHelpCommand(chatID int64) {
	helpText := `🤖 *Бот-интервьюер с анализом профиля*

*Команды:*
/start - Начать новое интервью
/status - Проверить прогресс текущего интервью
/restart - Перезапустить интервью
/stop - Остановить текущее интервью
/help - Показать это сообщение

*Как это работает:*
1. Используйте /start для начала интервью
2. Отвечайте на вопросы максимально честно и подробно
3. Интервью состоит из %d блоков
4. В каждом блоке до %d вопросов
5. После завершения создается психологический профиль

*🧠 Анализ профиля:*
• Автоматический анализ ваших ответов
• Выявление ценностей и мотиваций
• Анализ семейных паттернов
• Карьерные ориентации
• Способы преодоления трудностей

*Совет:* Чем подробнее ваши ответы, тем точнее будет анализ!`

	maxQuestions := h.config.GetQuestionsPerBlock() + h.config.GetMaxFollowupQuestions()
	h.bot.SendFormattedMessage(chatID, helpText, h.config.GetTotalBlocks(), maxQuestions)
}

// handleStatusCommand показывает статус интервью
func (h *Handler) handleStatusCommand(chatID int64, session *UserSession) {
	switch session.State {
	case StateIdle:
		h.bot.SendMessage(chatID, "Интервью не начато. Используйте /start для начала.")
	case StateInterview, StateWaitingAnswer:
		progress := fmt.Sprintf("📊 *Прогресс интервью*\n\n"+
			"🆔 ID: `%s`\n"+
			"📋 Блок: %d/%d (%s)\n"+
			"❓ Вопросов в блоке: %d\n"+
			"⏰ Состояние: %s",
			session.InterviewID,
			session.CurrentBlock, h.config.GetTotalBlocks(),
			h.getCurrentBlockTitle(session.CurrentBlock),
			session.QuestionCount,
			h.getStateDescription(session.State))
		h.bot.SendMessage(chatID, progress)
	case StateCompleted:
		h.bot.SendFormattedMessage(chatID, "✅ Интервью завершено!\n🆔 ID: `%s`", session.InterviewID)
	}
}

// handleRestartCommand перезапускает интервью
func (h *Handler) handleRestartCommand(chatID int64, session *UserSession) {
	h.resetSession(session)
	h.bot.SendMessage(chatID, "🔄 Интервью сброшено. Используйте /start для начала нового интервью.")
}

// handleStopCommand останавливает интервью
func (h *Handler) handleStopCommand(chatID int64, session *UserSession) {
	if session.State == StateIdle {
		h.bot.SendMessage(chatID, "Интервью не запущено.")
		return
	}

	h.resetSession(session)
	h.bot.SendMessage(chatID, "🛑 Интервью остановлено.")
}

// Улучшенная валидация пользовательского ввода
func (h *Handler) validateUserInput(text string) error {
	if len(text) > 4000 {
		return fmt.Errorf("сообщение слишком длинное (максимум 4000 символов)")
	}

	// Проверка на спам/повторяющиеся символы
	if strings.Count(text, text[:1]) > len(text)*1 && len(text) > 10 {
		return fmt.Errorf("сообщение содержит слишком много повторяющихся символов")
	}

	return nil
}

// Обновленный handleUserInput
func (h *Handler) handleUserInput(chatID int64, text string, session *UserSession) {
	if session.State != StateWaitingAnswer {
		h.bot.SendMessage(chatID, "Сейчас не время для ответов. Используйте /start для начала интервью или /help для помощи.")
		return
	}

	// Валидация ввода
	if err := h.validateUserInput(text); err != nil {
		h.bot.SendMessage(chatID, "❌ "+err.Error())
		return
	}

	h.processUserAnswer(chatID, text, session)
}

// initializeInterview инициализирует новое интервью
func (h *Handler) initializeInterview(chatID int64, session *UserSession) {
	// Сбрасываем сессию
	h.resetSession(session)

	// Создаем новое интервью
	session.InterviewID = uuid.New().String()
	session.State = StateInterview
	session.CurrentBlock = 1
	session.QuestionCount = 0
	session.Result = &storage.InterviewResult{
		InterviewID: session.InterviewID,
		Timestamp:   time.Now().Format(time.RFC3339),
		Blocks:      make([]storage.BlockResult, 0, h.config.GetTotalBlocks()),
	}

	// Отправляем приветствие
	welcomeText := fmt.Sprintf(`🎯 *Добро пожаловать в интервью!*

🆔 *ID интервью:* `+"`%s`"+`
📋 *Всего блоков:* %d
❓ *Вопросов в блоке:* до %d
⏱ *Время:* ~%d минут

*Правила:*
• Отвечайте честно и подробно
• Можете отвечать в несколько сообщений
• Используйте /status для проверки прогресса
• Используйте /stop для остановки

Готовы начать? Сейчас начнется первый блок! 🚀`,
		session.InterviewID,
		h.config.GetTotalBlocks(),
		h.config.GetQuestionsPerBlock()+h.config.GetMaxFollowupQuestions(),
		h.config.GetTotalBlocks()*3)

	h.bot.SendMessage(chatID, welcomeText)

	// Начинаем первый блок
	h.startNextBlock(chatID, session)
}

// processUserAnswer обрабатывает ответ пользователя
func (h *Handler) processUserAnswer(chatID int64, answer string, session *UserSession) {
	// Добавляем ответ в текущий диалог (последний вопрос)
	if len(session.CurrentDialogue) > 0 {
		lastIndex := len(session.CurrentDialogue) - 1
		session.CurrentDialogue[lastIndex].Answer = answer
	}

	session.QuestionCount++
	maxQuestions := h.config.GetQuestionsPerBlock() + h.config.GetMaxFollowupQuestions()

	// Проверяем, нужен ли следующий вопрос в блоке
	if session.QuestionCount < maxQuestions {
		// Генерируем следующий вопрос
		h.generateNextQuestion(chatID, session)
	} else {
		// Завершаем блок
		h.finishCurrentBlock(chatID, session)
	}
}

// generateNextQuestion генерирует следующий вопрос
func (h *Handler) generateNextQuestion(chatID int64, session *UserSession) {
	h.bot.SendMessage(chatID, "🤔 Генерирую следующий вопрос...")

	block := h.config.Blocks[session.CurrentBlock-1]

	// Вызываем интервьюера для получения следующего вопроса
	question, err := h.interviewer.GenerateQuestion(block, session.CurrentDialogue, session.CumulativeSummaries, h.config)
	if err != nil {
		h.bot.SendMessage(chatID, "Произошла ошибка при генерации вопроса. Попробуйте еще раз.")
		return
	}

	// Проверяем, завершает ли AI блок
	if h.isBlockComplete(question) {
		h.finishCurrentBlock(chatID, session)
		return
	}

	// Добавляем вопрос в диалог
	session.CurrentDialogue = append(session.CurrentDialogue, storage.QA{
		Question: question,
		Answer:   "", // Будет заполнен при получении ответа
	})

	session.State = StateWaitingAnswer
	h.bot.SendFormattedMessage(chatID, "❓ *Вопрос %d:*\n\n%s", session.QuestionCount+1, question)
}

// startNextBlock начинает следующий блок
func (h *Handler) startNextBlock(chatID int64, session *UserSession) {
	if session.CurrentBlock > h.config.GetTotalBlocks() {
		h.completeInterview(chatID, session)
		return
	}

	block := h.config.Blocks[session.CurrentBlock-1]
	session.QuestionCount = 0
	session.CurrentDialogue = []storage.QA{}

	// Отправляем информацию о блоке
	blockInfo := fmt.Sprintf("📋 *Блок %d/%d: %s*\n\nСейчас мы поговорим о %s",
		session.CurrentBlock, h.config.GetTotalBlocks(), block.Title, strings.ToLower(block.Title))

	h.bot.SendMessage(chatID, blockInfo)

	// Генерируем первый вопрос блока
	h.generateNextQuestion(chatID, session)
}

// Вспомогательные методы
func (h *Handler) getOrCreateSession(userID int64) *UserSession {
	h.sessionsMutex.Lock()
	defer h.sessionsMutex.Unlock()

	if session, exists := h.sessions[userID]; exists {
		return session
	}

	session := &UserSession{
		UserID: userID,
		State:  StateIdle,
	}
	h.sessions[userID] = session
	return session
}

func (h *Handler) resetSession(session *UserSession) {
	session.State = StateIdle
	session.CurrentBlock = 0
	session.QuestionCount = 0
	session.CurrentDialogue = []storage.QA{}
	session.CumulativeSummaries = []string{}
	session.Result = nil
	session.InterviewID = ""
}

func (h *Handler) getCurrentBlockTitle(blockNum int) string {
	if blockNum <= 0 || blockNum > len(h.config.Blocks) {
		return "Неизвестный блок"
	}
	return h.config.Blocks[blockNum-1].Title
}

func (h *Handler) getStateDescription(state SessionState) string {
	switch state {
	case StateIdle:
		return "Ожидание"
	case StateInterview:
		return "Интервью"
	case StateWaitingAnswer:
		return "Ожидание ответа"
	case StateCompleted:
		return "Завершено"
	default:
		return "Неизвестно"
	}
}

func (h *Handler) isBlockComplete(question string) bool {
	lowerQuestion := strings.ToLower(question)
	return strings.Contains(lowerQuestion, "завершаем") ||
		strings.Contains(lowerQuestion, "переходим") ||
		strings.Contains(lowerQuestion, "блок завершен") ||
		strings.Contains(lowerQuestion, "следующий блок")
}

func (h *Handler) finishCurrentBlock(chatID int64, session *UserSession) {
	h.bot.SendMessage(chatID, "📝 Обрабатываю блок...")

	block := h.config.Blocks[session.CurrentBlock-1]

	// Создаем результат блока
	blockResult := &storage.BlockResult{
		BlockID:             block.ID,
		BlockName:           block.Name,
		QuestionsAndAnswers: session.CurrentDialogue,
	}

	// Создаем саммари
	summary, err := h.interviewer.CreateSummary(session.CurrentDialogue, h.config)
	if err != nil {
		h.bot.SendMessage(chatID, "Ошибка при создании саммари блока.")
		return
	}

	// Добавляем результат и саммари
	session.Result.Blocks = append(session.Result.Blocks, *blockResult)
	session.CumulativeSummaries = append(session.CumulativeSummaries, summary)

	// Информируем о завершении блока
	h.bot.SendFormattedMessage(chatID, "✅ Блок %d завершен! Переходим к следующему...", session.CurrentBlock)

	// Переходим к следующему блоку
	session.CurrentBlock++
	h.startNextBlock(chatID, session)
}

func (h *Handler) completeInterview(chatID int64, session *UserSession) {
	// Сохраняем результат интервью
	err := storage.SaveResult(session.Result)
	if err != nil {
		h.bot.SendMessage(chatID, "Ошибка сохранения результата интервью.")
		return
	}

	session.State = StateCompleted

	// Начинаем анализ профиля
	h.bot.SendMessage(chatID, "🎉 Интервью завершено! Начинаю анализ вашего психологического профиля...")

	// Извлекаем профиль с помощью Profile Extractor
	if h.extractor != nil {
		go h.processProfileExtraction(chatID, session)
	}

	completionText := fmt.Sprintf(`✅ *Интервью успешно завершено!*

📊 Собрано данных:
• %d блоков пройдено
• %d ответов получено
• 🆔 ID: `+"`%s`"+`

🧠 Анализ профиля в процессе...
Результат будет готов через 1-2 минуты.

Используйте /start для нового интервью.`,
		len(session.Result.Blocks),
		h.getTotalAnswersCount(session.Result),
		session.InterviewID)

	h.bot.SendMessage(chatID, completionText)
}

// processProfileExtraction обрабатывает извлечение профиля в отдельной горутине
func (h *Handler) processProfileExtraction(chatID int64, session *UserSession) {
	profileResult, err := h.extractor.ExtractProfile(session.Result)
	if err != nil {
		h.bot.SendMessage(chatID, "❌ Ошибка при анализе профиля: "+err.Error())
		return
	}

	if !profileResult.Success {
		h.bot.SendMessage(chatID, "❌ Не удалось проанализировать профиль: "+profileResult.Error)
		return
	}

	// Сохраняем профиль в файл
	fileName, err := h.extractor.SaveProfile(session.InterviewID, profileResult)
	if err != nil {
		h.bot.SendMessage(chatID, "⚠️ Профиль создан, но не удалось сохранить файл: "+err.Error())
	}

	// Получаем краткое резюме для Telegram
	summary, err := h.extractor.GetProfileSummary(profileResult.ProfileJSON)
	if err != nil {
		summary = "Профиль создан, но не удалось сгенерировать резюме."
	}

	// Отправляем результат пользователю
	resultMessage := fmt.Sprintf(`🎯 *Анализ профиля завершен!*

%s

💾 Полный профиль сохранен в: `+"`%s`"+`

🔍 Профиль содержит детальный анализ:
• Семейные паттерны и влияния
• Ценностные ориентации  
• Карьерные мотивации
• Способы преодоления трудностей
• Планы на будущее

_Этот анализ создан искусственным интеллектом на основе ваших ответов._`,
		summary,
		fileName)

	h.bot.SendMessage(chatID, resultMessage)
}

func (h *Handler) getTotalAnswersCount(result *storage.InterviewResult) int {
	count := 0
	for _, block := range result.Blocks {
		count += len(block.QuestionsAndAnswers)
	}
	return count
}

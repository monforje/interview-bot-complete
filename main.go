package main

import (
	"fmt"
	"interview-bot-complete/internal/config"
	"interview-bot-complete/internal/extractor"
	"interview-bot-complete/internal/interviewer"
	"interview-bot-complete/internal/telegram"
	"log"
	"os"

	"github.com/joho/godotenv"
)

func main() {
	fmt.Println("🚀 Запуск Interview Bot с Profile Extractor...")

	// Загружаем переменные окружения
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Ошибка загрузки .env файла")
	}

	// Проверяем наличие API ключей
	openaiKey := os.Getenv("OPENAI_API_KEY")
	if openaiKey == "" {
		log.Fatal("OPENAI_API_KEY не установлен в .env файле")
	}

	telegramToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if telegramToken == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN не установлен в .env файле")
	}

	// Загружаем конфигурацию интервью
	cfg, err := config.Load("config/interview.yaml")
	if err != nil {
		log.Fatalf("Ошибка загрузки конфигурации интервью: %v", err)
	}

	// Инициализируем сервисы
	fmt.Println("🔧 Инициализация сервисов...")

	// Интервьюер для Telegram бота
	interviewerService := interviewer.New(openaiKey)
	fmt.Println("✅ Интервьюер инициализирован")

	// Profile Extractor для анализа
	extractorService, err := extractor.New(openaiKey)
	if err != nil {
		log.Printf("⚠️ Ошибка инициализации Profile Extractor: %v", err)
		log.Println("Бот будет работать без анализа профилей")
		extractorService = nil
	} else {
		fmt.Println("✅ Profile Extractor инициализирован")
	}

	// Telegram бот
	bot := telegram.New(telegramToken)
	handler := telegram.NewHandler(bot, cfg, interviewerService, extractorService)
	fmt.Println("✅ Telegram бот инициализирован")

	// Выводим информацию о конфигурации
	fmt.Println("\n📋 Конфигурация:")
	fmt.Printf("• Блоков в интервью: %d\n", cfg.GetTotalBlocks())
	fmt.Printf("• Вопросов на блок: до %d\n", cfg.GetQuestionsPerBlock()+cfg.GetMaxFollowupQuestions())

	if extractorService != nil {
		fmt.Println("• Анализ профилей: включен 🧠")
	} else {
		fmt.Println("• Анализ профилей: отключен ⚠️")
	}

	fmt.Println("\n🤖 Telegram бот запущен!")
	fmt.Println("⏳ Ожидание сообщений...")
	fmt.Println("📱 Найдите бота в Telegram и отправьте /start")

	// Запускаем polling
	err = bot.StartPolling(handler.HandleUpdate)
	if err != nil {
		log.Fatalf("Ошибка запуска бота: %v", err)
	}
}

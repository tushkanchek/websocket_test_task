package main

import (
	"context"
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"go-ws-chat/internal/config"
	"go-ws-chat/internal/handler"
	"go-ws-chat/internal/infra"
	"go-ws-chat/internal/service"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time" // Импорт для таймаута Health Check
)



func main() {
	// Настройка логгирования
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// 1. Загрузка конфигурации и контекста
	cfg := config.LoadConfig() 
	ctx, cancel := context.WithCancel(context.Background()) 

	// 2. Инициализация всех зависимостей и Fiber
	app, broker, _ := setupDependencies(cfg, ctx, logger)

	// 3. Запуск сервера и ожидание сигналов завершения
	runServer(app, cfg, broker, cancel, logger)
}




// HealthCheckHandler возвращает обработчик Fiber, который проверяет доступность Broker (Redis).
func HealthCheckHandler(broker infra.Broker, logger *slog.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Устанавливаем таймаут для проверки Redis
		ctx, cancel := context.WithTimeout(c.Context(), 2*time.Second)
		defer cancel()

		// Проверяем соединение с Redis через Ping
		if err := broker.Ping(ctx); err != nil {
			logger.Error("Health check failed: Redis is unreachable", slog.String("error", err.Error()))
			// Возвращаем 503 Service Unavailable, если Redis недоступен
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"status": "fail",
				"service": "Redis",
			})
		}

		// Если все хорошо
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"status": "ok",
			"service": "Chat-API",
		})
	}
}

// setupDependencies инициализирует все слои приложения.
func setupDependencies(cfg *config.Config, ctx context.Context, logger *slog.Logger) (*fiber.App, infra.Broker, *service.Hub) {
	// 1. Инициализация инфраструктуры
	broker := infra.NewRedisBroker(cfg.RedisAddr, cfg.WSStream, cfg.WSGroup)
	
	// 2. Инициализация сервисов (Hub)
	hub := service.NewHub(broker, ctx) 
	
	// 3. Настройка Fiber
	app := fiber.New()
	
	// 4. Регистрация маршрутов
	registerRoutes(app, broker, hub, logger)

	return app, broker, hub
}

// registerRoutes регистрирует все HTTP/WS маршруты
func registerRoutes(app *fiber.App, broker infra.Broker, hub *service.Hub, logger *slog.Logger) {
	// Healthcheck
	app.Get("/health", HealthCheckHandler(broker, logger)) 
	
	// Регистрация WebSocket маршрутов
	wsHandler := handler.NewWSHandler(hub)
	app.Use("/ws", wsHandler.UpgradeMiddleware)
	app.Get("/ws", websocket.New(wsHandler.HandleWebSocket))
}

// gracefulShutdown корректно закрывает все ресурсы приложения
func gracefulShutdown(app *fiber.App, broker infra.Broker, cancel context.CancelFunc, logger *slog.Logger) {
	// 1. Отменяем контекст, останавливая Consumer
	cancel() 
	logger.Info("1. Graceful shutdown: Context cancelled for consumers.")
	
	// 2. Корректно закрываем Fiber
	if err := app.Shutdown(); err != nil {
		logger.Error("Fiber Shutdown Error", "error", err)
	}
	logger.Info("2. Graceful shutdown: HTTP/WS server stopped.")
	
	// 3. Закрываем соединение с Redis
	if err := broker.Close(); err != nil {
		logger.Error("Error closing Redis connection", "error", err)
	}
	logger.Info("3. Graceful shutdown: Redis connection closed.")
}

// runServer запускает Fiber и ожидает сигнала завершения
func runServer(app *fiber.App, cfg *config.Config, broker infra.Broker, cancel context.CancelFunc, logger *slog.Logger) {
	// Запуск сервера в отдельной горутине
	go func() {
		logger.Info("Starting Fiber Server", "port", cfg.AppPort)
		if err := app.Listen(":" + cfg.AppPort); err != nil && err != http.ErrServerClosed {
			logger.Error("Fiber server error", "error", err)
		}
	}()
	
	// Блокируем main-горутину до получения сигнала завершения
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM) 
	<-quit 

	logger.Info("Received shutdown signal. Starting graceful shutdown.")
	
	// Запускаем последовательное закрытие ресурсов
	gracefulShutdown(app, broker, cancel, logger)
	logger.Info("Application gracefully stopped.")
}



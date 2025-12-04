package main
import (
	"context"
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"go-ws-chat/internal/config"
	"go-ws-chat/internal/handler"
	"go-ws-chat/internal/infra"
	"go-ws-chat/internal/service"
	"log/slog" // Структурированное логирование
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// Настройка структурированного логгирования (JSON)
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// 1. Загрузка конфигурации
	cfg := config.LoadConfig() 
    
    // 2. Создаем контекст для Graceful Shutdown
    ctx, cancel := context.WithCancel(context.Background()) 

	// 3. Инициализация инфраструктуры и сервисов
	broker := infra.NewRedisBroker(cfg.RedisAddr, cfg.WSStream, cfg.WSGroup)
	// Передаем контекст Hub, чтобы он запустил Consumer с возможностью отмены
	hub := service.NewHub(broker, ctx) 
	wsHandler := handler.NewWSHandler(hub)

	// 4. Настройка Fiber
	app := fiber.New()

	// Healthcheck для мониторинга
	app.Get("/health", func(c *fiber.Ctx) error {
	    return c.JSON(fiber.Map{"status": "ok"})
	})

	app.Use("/ws", wsHandler.UpgradeMiddleware)
	app.Get("/ws", websocket.New(wsHandler.HandleWebSocket)) 

	// 5. Запуск сервера в отдельной горутине
	go func() {
		slog.Info("Starting Fiber Server", "port", cfg.AppPort)
		if err := app.Listen(":" + cfg.AppPort); err != nil && err != http.ErrServerClosed {
			slog.Error("Fiber server error", "error", err)
		}
	}()
    
	// 6. Реализация Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM) 
    
	<-quit 
	slog.Info("Received shutdown signal. Starting graceful shutdown.")

	// a) Отменяем контекст, останавливая Consumer
	cancel() 
	slog.Info("1. Graceful shutdown: Context cancelled for consumers.")
    
	// b) Корректно закрываем Fiber
	if err := app.Shutdown(); err != nil {
		slog.Error("Fiber Shutdown Error", "error", err)
	}
	slog.Info("2. Graceful shutdown: HTTP/WS server stopped.")
    
	// c) Закрываем соединение с Redis
	if err := broker.Close(); err != nil {
	    slog.Error("Error closing Redis connection", "error", err)
	}
	slog.Info("3. Graceful shutdown: Redis connection closed.")

	slog.Info("Application gracefully stopped.")
}
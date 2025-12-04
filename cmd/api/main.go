package main

import (
	"context"
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"go-ws-chat/internal/config"
	"go-ws-chat/internal/handler"
	"go-ws-chat/internal/infra"
	"go-ws-chat/internal/service"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// 1. Загрузка конфигурации
	cfg := config.LoadConfig() 
    
    // 2. Создаем контекст для Graceful Shutdown (управления Consumer)
    ctx, cancel := context.WithCancel(context.Background()) 

	// 3. Инициализация инфраструктуры и сервисов (Dependency Injection)
	broker := infra.NewRedisBroker(cfg.RedisAddr, cfg.WSStream, cfg.WSGroup)
	// Передаем контекст Hub, чтобы он запустил Consumer с возможностью отмены
	hub := service.NewHub(broker, ctx) 
	wsHandler := handler.NewWSHandler(hub)

	// 4. Настройка Fiber
	app := fiber.New()
	app.Use(logger.New())

	app.Use("/ws", wsHandler.UpgradeMiddleware)
	// Обертка websocket.New() устраняет ошибку несовместимости типов
	app.Get("/ws", websocket.New(wsHandler.HandleWebSocket)) 

	// 5. Запуск сервера в отдельной горутине
	go func() {
		log.Printf("Starting Fiber Server on :%s", cfg.AppPort)
		if err := app.Listen(":" + cfg.AppPort); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Fiber server error: %v", err)
		}
	}()
    
	// 6. Реализация Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM) 
    
	<-quit // Блокируем до получения сигнала
	log.Println("--- Received shutdown signal. Starting graceful shutdown. ---")

	// a) Отменяем контекст, останавливая потребительские горутины
	cancel() 
	log.Println("1. Graceful shutdown: Context cancelled for consumers.")
    
	// b) Корректно закрываем Fiber
	if err := app.Shutdown(); err != nil {
		log.Fatalf("Fiber Shutdown Error: %v", err)
	}
	log.Println("2. Graceful shutdown: HTTP/WS server stopped.")
    
	// c) Закрываем соединение с Redis
	if err := broker.Close(); err != nil {
	    log.Printf("Error closing Redis connection: %v", err)
	}
	log.Println("3. Graceful shutdown: Redis connection closed.")

	log.Println("Application gracefully stopped.")
}
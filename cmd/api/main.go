package main

import (
	"go-ws-chat/internal/config" // 
	"go-ws-chat/internal/handler"
	"go-ws-chat/internal/infra"
	"go-ws-chat/internal/service"
	"log"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

func main() {
	// 1. Загрузка конфигурации
	cfg := config.LoadConfig() //

	// 2. Инициализация инфраструктуры
	broker := infra.NewRedisBroker(cfg.RedisAddr, cfg.WSStream, cfg.WSGroup)
	
	// 3. Инициализация сервисного слоя
	hub := service.NewHub(broker)
	
	// 4. Инициализация транспортного слоя
	wsHandler := handler.NewWSHandler(hub)

	// 5. Настройка Fiber
	app := fiber.New()
	app.Use(logger.New())

	// Роуты
	app.Use("/ws", wsHandler.UpgradeMiddleware)
	app.Get("/ws", websocket.New(wsHandler.HandleWebSocket))

	// 6. Запуск сервера с использованием порта из конфига
	log.Printf("Starting Server on :%s", cfg.AppPort)
	log.Fatal(app.Listen(":" + cfg.AppPort))
}
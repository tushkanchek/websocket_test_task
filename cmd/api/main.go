package main

import (
	"go-ws-chat/internal/handler"
	"go-ws-chat/internal/infra"
	"go-ws-chat/internal/service"
	"log"
	"os"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

func main() {
	// Получаем переменные окружения
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "redis:6379"
	}
	streamName := os.Getenv("WS_STREAM_NAME")
	if streamName == "" {
		streamName = "chat_stream"
	}
	groupName := os.Getenv("WS_GROUP_NAME")
	if groupName == "" {
		groupName = "chat_group"
	}

	// 1. Инициализация инфраструктуры (Redis Broker)
	broker := infra.NewRedisBroker(redisAddr, streamName, groupName)
	
	// 2. Инициализация сервисного слоя (Hub), внедрение брокера
	hub := service.NewHub(broker)
	
	// 3. Инициализация транспортного слоя (Handler), внедрение Hub
	wsHandler := handler.NewWSHandler(hub)

	// 4. Настройка Fiber
	app := fiber.New()
	app.Use(logger.New())

	// Роуты
	app.Use("/ws", wsHandler.UpgradeMiddleware)
	app.Get("/ws", websocket.New(wsHandler.HandleWebSocket))	

	// 5. Запуск сервера
	log.Println("Starting Server on :3000")
	log.Fatal(app.Listen(":3000"))
}
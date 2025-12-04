package handler

import (
	"context"
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"go-ws-chat/internal/entity"
	"go-ws-chat/internal/service"
	"time"
	"log"
)

type WSHandler struct {
	hub *service.Hub
}

func NewWSHandler(hub *service.Hub) *WSHandler {
	return &WSHandler{hub: hub}
}

// UpgradeMiddleware проверяет, что это запрос на апгрейд до WebSocket
func (h *WSHandler) UpgradeMiddleware(c *fiber.Ctx) error {
	if websocket.IsWebSocketUpgrade(c) {
		return c.Next()
	}
	return fiber.ErrUpgradeRequired // 426
}

// HandleWebSocket — основной цикл обработки соединения
func (h *WSHandler) HandleWebSocket(c *websocket.Conn) {
	// 1. Авторизация по токену
	// Токен из query params: ws://localhost:3000/ws?token={USER_ID}
	userID := c.Query("token")
	if userID == "" {
		c.WriteJSON(fiber.Map{"error": "Unauthorized: token required"})
		c.Close()
		log.Println("[Handler] Connection rejected: No token")
		return
	}

	// 2. Регистрация в хабе
	h.hub.Register(userID, c)
	defer h.hub.Unregister(userID)

	// Цикл чтения сообщений от клиента
	for {
		var msg entity.Message
		if err := c.ReadJSON(&msg); err != nil {
			log.Printf("[Handler] Read error for %s: %v", userID, err)
			break // Ошибка чтения или разрыв соединения
		}

		// Заполняем системные поля
		msg.FromID = userID
		msg.Timestamp = time.Now().Unix()

		// 3. Публикация в очередь
		if err := h.hub.Broadcast(context.Background(), msg); err != nil {
			log.Printf("[Handler] Failed to publish message: %v", err)
			// Здесь можно обработать ошибку: отправить клиенту сообщение об ошибке или закрыть соединение
		}
	}
}
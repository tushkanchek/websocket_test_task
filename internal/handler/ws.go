package handler

import (
	"context"
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"go-ws-chat/internal/entity"
	"go-ws-chat/internal/service"
	"log"
	"time"
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
	return fiber.ErrUpgradeRequired
}

// HandleWebSocket — основной цикл обработки соединения
func (h *WSHandler) HandleWebSocket(c *websocket.Conn) {
	userID := c.Query("token")
	if userID == "" {
		c.WriteJSON(fiber.Map{"error": "Unauthorized: token required"})
		c.Close()
		log.Println("[Handler] Connection rejected: No token")
		return
	}

	h.hub.Register(userID, c)
	defer h.hub.Unregister(userID)

	// Цикл чтения сообщений от клиента
	for {
		var msg entity.Message
		if err := c.ReadJSON(&msg); err != nil {
			log.Printf("[Handler] Read error for %s: %v", userID, err)
			break 
		}

		msg.FromID = userID
		msg.Timestamp = time.Now().Unix()

		// Публикация в очередь с тайм-аутом (Крутое использование контекста)
		// Если Redis не ответит за 5 секунд, операция прервется.
		publishCtx, cancel := context.WithTimeout(context.Background(), 5 * time.Second)
		
		if err := h.hub.Broadcast(publishCtx, msg); err != nil {
			log.Printf("[Handler] Failed to publish message (Error: %v)", err)
		}
		cancel() 
	}
}
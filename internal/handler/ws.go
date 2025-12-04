package handler

import (
	"context"
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"go-ws-chat/internal/entity"
	"go-ws-chat/internal/service"
	"log/slog"
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
		slog.Warn("Connection rejected: No token", "component", "ws_handler")
		return
	}

	h.hub.Register(userID, c)
	defer h.hub.Unregister(userID)

	for {
		var msg entity.Message
		if err := c.ReadJSON(&msg); err != nil {
			slog.Info("Client read error, closing connection", "user_id", userID, "error", err)
			break 
		}

		msg.FromID = userID
		msg.Timestamp = time.Now().Unix()

		// Публикация в очередь с таймаутом (5 секунд)
		publishCtx, cancel := context.WithTimeout(context.Background(), 5 * time.Second)
		
		if err := h.hub.Broadcast(publishCtx, msg); err != nil {
			slog.Error("Failed to publish message", "error", err, "user_id", userID)
		}
		cancel() // Освобождаем ресурсы контекста
	}
}
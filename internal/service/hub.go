package service

import (
	"context"
	"github.com/gofiber/contrib/websocket"
	"go-ws-chat/internal/entity"
	"go-ws-chat/internal/infra"
	"log"
	"sync"
)

// Hub управляет соединениями
type Hub struct {
	clients map[string]*websocket.Conn
	mu      sync.RWMutex
	broker  infra.Broker
}

// NewHub принимает контекст Graceful Shutdown
func NewHub(broker infra.Broker, ctx context.Context) *Hub {
	h := &Hub{
		clients: make(map[string]*websocket.Conn),
		broker:  broker,
	}
	
	// Запускаем Consumer, используя контекст, который будет отменен в main.go
	go h.startConsumer(ctx)
	return h
}

// Register сохраняет соединение пользователя
func (h *Hub) Register(userID string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[userID] = conn
	log.Printf("[HUB] User %s connected", userID)
}

// Unregister удаляет соединение
func (h *Hub) Unregister(userID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.clients[userID]; ok {
		delete(h.clients, userID)
		log.Printf("[HUB] User %s disconnected", userID)
	}
}

// Broadcast принимает контекст из Handler и передает его дальше
func (h *Hub) Broadcast(ctx context.Context, msg entity.Message) error {
	return h.broker.Publish(ctx, msg)
}

// startConsumer слушает очередь, используя переданный контекст для остановки
func (h *Hub) startConsumer(ctx context.Context) {
	h.broker.Consume(ctx, func(msg entity.Message) {
		h.deliverToUser(msg)
	})
}

// deliverToUser ищет пользователя в локальной памяти и пишет в сокет
func (h *Hub) deliverToUser(msg entity.Message) {
	h.mu.RLock()
	client, ok := h.clients[msg.ToID]
	h.mu.RUnlock()

	if ok {
		if err := client.WriteJSON(msg); err != nil {
			log.Printf("[HUB] Error writing to client %s: %v. Closing connection.", msg.ToID, err)
			client.Close()
			h.Unregister(msg.ToID)
		}
	}
}
package entity

// Message — это основная сущность приложения.
// Она представляет собой сообщение, которое публикуется в очередь и доставляется клиенту.
// Структура не зависит от WebSocket (транспорт) или Redis (инфраструктура).
type Message struct {
	FromID    string `json:"from_id"`    // Отправитель (UserID)
	ToID      string `json:"to_id"`      // Получатель (UserID)
	Content   string `json:"content"`    // Тело сообщения
	Timestamp int64  `json:"timestamp"`  // Время отправки
}
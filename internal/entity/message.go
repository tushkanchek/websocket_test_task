package entity

// Message — это основная сущность приложения.
type Message struct {
	FromID    string `json:"from_id"`
	ToID      string `json:"to_id"`
	Content   string `json:"content"`
	Timestamp int64  `json:"timestamp"`
}
package models

import "time"

// Полезная нагрузка события "new_message", которое шлем по WebSocket.
type WSNewMessageEvent struct {
	Type    string           `json:"type"`
	ChatID  int              `json:"chatId"`
	Message WSMessagePayload `json:"message"`
}

// Структура сообщения для ws, по сути зеркалит ChatMessage.
type WSMessagePayload struct {
	ID         int       `json:"id"`
	ChatID     int       `json:"chatId"`
	SenderID   int       `json:"senderId"`
	Type       string    `json:"type"`
	ContractID *int      `json:"contractId,omitempty"`
	Status     *string   `json:"status,omitempty"`
	Text       string    `json:"text"`
	CreatedAt  time.Time `json:"createdAt"`
	IsSystem   *bool     `json:"isSystem,omitempty"`
}

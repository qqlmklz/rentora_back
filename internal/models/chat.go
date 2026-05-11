package models

import "time"

// Тело запроса для POST /api/chats (propertyId обязателен и > 0).
type CreateChatRequest struct {
	PropertyID *int `json:"propertyId"`
}

// Успешный ответ POST /api/chats (для кнопки «Написать» в объявлении).
type CreateChatResponse struct {
	ChatID  int `json:"chatId"`
	OwnerID int `json:"ownerId"` // идентификатор владельца объявления (если сравнить с currentUser.id, можно понять роль арендодателя)
}

// Одна запись списка для GET /api/chats (companion* — это собеседник, не текущий пользователь).
type ChatListItem struct {
	ID              int        `json:"id"`
	PropertyID      int        `json:"propertyId"`
	OwnerID         int        `json:"ownerId"` // владелец объявления (properties.user_id), в чате это тот же seller
	PropertyTitle   string     `json:"propertyTitle"`
	PropertyPhoto   *string    `json:"propertyPhoto"`
	CompanionID     int        `json:"companionId"`
	CompanionName   string     `json:"companionName"`
	CompanionAvatar *string    `json:"companionAvatar"`
	LastMessage     *string    `json:"lastMessage"`
	LastMessageAt   *time.Time `json:"lastMessageAt"`
	UnreadCount     int        `json:"unreadCount"`
}

// Ответ для GET /api/chats/:id (шапка чата без ленты сообщений).
type ChatDetailResponse struct {
	ID              int     `json:"id"`
	PropertyID      int     `json:"propertyId"`
	PropertyTitle   string  `json:"propertyTitle"`
	PropertyOwnerID int     `json:"propertyOwnerId"`
	CompanionName   string  `json:"companionName"`
	CompanionAvatar *string `json:"companionAvatar"`
}

// Ответ для GET /api/chats/:id/messages (шапка + лента сообщений).
type ChatMessagesResponse struct {
	PropertyID      int           `json:"propertyId"`
	PropertyTitle   string        `json:"propertyTitle"`
	PropertyOwnerID int           `json:"propertyOwnerId"`
	CompanionID     int           `json:"companionId"`
	CompanionName   string        `json:"companionName"`
	CompanionAvatar *string       `json:"companionAvatar"`
	Messages        []ChatMessage `json:"messages"`
}

// Тело запроса для POST /api/chats/:id/messages.
type SendMessageRequest struct {
	Text string `json:"text" binding:"required"`
}

// Одна запись сообщения в GET /api/chats/:id/messages и в ответе POST.
// Тип бывает "text" или "contract" (служебные сообщения идут как text + isSystem).
type ChatMessage struct {
	ID               int       `json:"id"`
	ChatID           int       `json:"chatId"`
	SenderID         int       `json:"senderId"`
	Type             string    `json:"type"`
	ContractID       *int      `json:"contractId,omitempty"`
	Status           *string   `json:"status,omitempty"` // статус договора, когда тип сообщения contract
	Text             string    `json:"text"`
	CreatedAt        time.Time `json:"createdAt"`
	IsSystem         *bool     `json:"isSystem,omitempty"`
}

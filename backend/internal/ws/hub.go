package ws

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"rentora/backend/internal/models"

	"github.com/gorilla/websocket"
)

// Хаб хранит активные ws-соединения по user id (несколько вкладок = несколько conn).
type Hub struct {
	mu    sync.RWMutex
	conns map[int]map[*websocket.Conn]struct{}
}

// Создаем пустой Hub.
func NewHub() *Hub {
	return &Hub{conns: make(map[int]map[*websocket.Conn]struct{})}
}

// Регистрируем новое соединение пользователя.
func (h *Hub) Register(userID int, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.conns[userID] == nil {
		h.conns[userID] = make(map[*websocket.Conn]struct{})
	}
	h.conns[userID][conn] = struct{}{}
	log.Printf("[ws] connected user_id=%d", userID)
}

// Удаляем соединение пользователя.
func (h *Hub) Unregister(userID int, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if m, ok := h.conns[userID]; ok {
		delete(m, conn)
		if len(m) == 0 {
			delete(h.conns, userID)
		}
	}
	log.Printf("[ws] disconnected user_id=%d", userID)
}

// Рассылаем событие new_message всем нужным пользователям (участникам чата).
func (h *Hub) BroadcastNewMessage(chatID, sellerID, buyerID int, msg models.WSMessagePayload) {
	ev := models.WSNewMessageEvent{
		Type:    "new_message",
		ChatID:  chatID,
		Message: msg,
	}
	payload, err := json.Marshal(ev)
	if err != nil {
		log.Printf("[ws] marshal new_message: %v", err)
		return
	}

	h.mu.RLock()
	var targets []*websocket.Conn
	seen := make(map[*websocket.Conn]struct{})
	for _, uid := range []int{sellerID, buyerID} {
		for c := range h.conns[uid] {
			if _, dup := seen[c]; !dup {
				seen[c] = struct{}{}
				targets = append(targets, c)
			}
		}
	}
	h.mu.RUnlock()

	for _, conn := range targets {
		_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
		if err := conn.WriteMessage(websocket.TextMessage, payload); err != nil {
			log.Printf("[ws] write user: %v", err)
			continue
		}
	}
	log.Printf("[ws] new_message broadcast chat_id=%d message_id=%d recipients=%d", chatID, msg.ID, len(targets))
}

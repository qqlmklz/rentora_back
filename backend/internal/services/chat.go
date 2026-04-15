package services

import (
	"context"
	"errors"
	"log"
	"strings"

	"rentora/backend/internal/models"
	"rentora/backend/internal/repository"
	"rentora/backend/internal/ws"
)

// Сервис чатов, которые привязаны к объявлениям.
//
// По безопасности: на /api/chats всегда нужен JWT, а список чатов показываем только участнику.
// Сообщения читаем/пишем только после IsChatParticipant; чужой chatId даем 403. При создании чата buyer всегда текущий пользователь.
type ChatService struct {
	repo *repository.DB
	hub  *ws.Hub
}

// Конструктор ChatService. hub может быть nil (например, в тестах), тогда ws-рассылка не работает.
func NewChatService(repo *repository.DB, hub *ws.Hub) *ChatService {
	return &ChatService{repo: repo, hub: hub}
}

// Эту ошибку отдаем, если пользователь пытается открыть чат по своему же объявлению.
var ErrChatSelf = errors.New("cannot chat with yourself about own listing")

// Эту ошибку отдаем, если после trim текст сообщения пустой.
var ErrEmptyMessage = errors.New("empty message")

// Эту ошибку отдаем, если сообщение длиннее лимита.
var ErrMessageTooLong = errors.New("message too long")

const maxMessageLen = 8000

// Создаем чат (покупатель = текущий пользователь, продавец = владелец объявления) или возвращаем уже существующий id.
func (s *ChatService) CreateOrGetChat(ctx context.Context, userID, propertyID int) (*models.CreateChatResponse, error) {
	if propertyID < 1 {
		return nil, repository.ErrPropertyNotFound
	}
	ownerID, err := s.repo.GetPropertyOwnerID(ctx, propertyID)
	if err != nil {
		return nil, err
	}
	if ownerID == userID {
		return nil, ErrChatSelf
	}
	chatID, err := s.repo.CreateOrGetChat(ctx, propertyID, ownerID, userID)
	if err != nil {
		return nil, err
	}
	return &models.CreateChatResponse{ChatID: chatID, OwnerID: ownerID}, nil
}

// Возвращаем чаты текущего пользователя вместе с собеседником и короткой инфой по объявлению.
func (s *ChatService) ListMyChats(ctx context.Context, userID int) ([]models.ChatListItem, error) {
	rows, err := s.repo.ListChatsForUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]models.ChatListItem, 0, len(rows))
	for _, r := range rows {
		out = append(out, models.ChatListItem{
			ID:              r.ChatID,
			PropertyID:      r.PropertyID,
			OwnerID:         r.PropertyOwnerID,
			PropertyTitle:   r.PropertyTitle,
			PropertyPhoto:   normalizeOptionalURL(r.PropertyPhoto),
			CompanionID:     r.CompanionID,
			CompanionName:   strings.TrimSpace(r.CompanionName),
			CompanionAvatar: normalizeOptionalURL(r.CompanionAvatar),
			LastMessage:     r.LastMessage,
			LastMessageAt:   r.LastMessageAt,
			UnreadCount:     r.UnreadCount,
		})
	}
	return out, nil
}

// Возвращаем объявление и собеседника для GET /api/chats/:id (только участнику чата).
func (s *ChatService) GetChatDetail(ctx context.Context, userID, chatID int) (*models.ChatDetailResponse, error) {
	meta, err := s.repo.GetChatSessionMeta(ctx, chatID, userID)
	if err != nil {
		return nil, err
	}
	return &models.ChatDetailResponse{
		ID:              meta.ChatID,
		PropertyID:      meta.PropertyID,
		PropertyTitle:   strings.TrimSpace(meta.PropertyTitle),
		PropertyOwnerID: meta.PropertyOwnerID,
		CompanionName:   strings.TrimSpace(meta.CompanionName),
		CompanionAvatar: normalizeOptionalURL(meta.CompanionAvatar),
	}, nil
}

// Возвращаем сообщения и профиль собеседника; заодно помечаем входящие как прочитанные.
func (s *ChatService) ListMessages(ctx context.Context, userID, chatID int) (*models.ChatMessagesResponse, error) {
	meta, err := s.repo.GetChatSessionMeta(ctx, chatID, userID)
	if err != nil {
		return nil, err
	}
	if err := s.repo.MarkIncomingMessagesRead(ctx, chatID, userID); err != nil {
		return nil, err
	}
	rows, err := s.repo.ListMessagesByChatID(ctx, chatID)
	if err != nil {
		return nil, err
	}
	out := make([]models.ChatMessage, 0, len(rows))
	for _, m := range rows {
		out = append(out, MessageRowToChatMessage(chatID, m))
	}
	return &models.ChatMessagesResponse{
		PropertyID:      meta.PropertyID,
		PropertyTitle:   strings.TrimSpace(meta.PropertyTitle),
		PropertyOwnerID: meta.PropertyOwnerID,
		CompanionID:     meta.CompanionID,
		CompanionName:   strings.TrimSpace(meta.CompanionName),
		CompanionAvatar: normalizeOptionalURL(meta.CompanionAvatar),
		Messages:        out,
	}, nil
}

// Пустую строку (или из пробелов) превращаем в nil, чтобы в JSON пришел null.
func normalizeOptionalURL(s *string) *string {
	if s == nil {
		return nil
	}
	t := strings.TrimSpace(*s)
	if t == "" {
		return nil
	}
	return &t
}

// Проверяем сообщение и сохраняем его в базе.
func (s *ChatService) SendMessage(ctx context.Context, userID, chatID int, text string) (*models.ChatMessage, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, ErrEmptyMessage
	}
	if len(text) > maxMessageLen {
		return nil, ErrMessageTooLong
	}
	ok, err := s.repo.IsChatParticipant(ctx, chatID, userID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, repository.ErrChatForbidden
	}
	m, err := s.repo.InsertMessage(ctx, chatID, userID, text, false)
	if err != nil {
		return nil, err
	}
	out := MessageRowToChatMessage(chatID, m)
	if s.hub != nil {
		sellerID, buyerID, err := s.repo.GetChatParticipantIDs(ctx, chatID)
		if err != nil {
			log.Printf("[chats] GetChatParticipantIDs for ws: %v", err)
		} else {
			s.hub.BroadcastNewMessage(chatID, sellerID, buyerID, WSMessagePayloadFromRow(chatID, m))
		}
	}
	return &out, nil
}

// Помечаем все входящие в чате как прочитанные (только если пользователь участник этого чата).
func (s *ChatService) MarkChatRead(ctx context.Context, userID, chatID int) error {
	ok, err := s.repo.IsChatParticipant(ctx, chatID, userID)
	if err != nil {
		return err
	}
	if !ok {
		return repository.ErrChatForbidden
	}
	return s.repo.MarkIncomingMessagesRead(ctx, chatID, userID)
}

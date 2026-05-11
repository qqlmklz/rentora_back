package services

import (
	"rentora/backend/internal/models"
	"rentora/backend/internal/repository"
)

// Перекладываем строку из БД в API-модель сообщения (type: text | contract).
func MessageRowToChatMessage(chatID int, m repository.MessageRow) models.ChatMessage {
	mt := m.MessageType
	if mt == "" {
		if m.IsSystem {
			mt = repository.MessageTypeSystem
		} else {
			mt = repository.MessageTypeText
		}
	}
	switch mt {
	case repository.MessageTypeContract:
		return models.ChatMessage{
			ID:         m.ID,
			ChatID:     chatID,
			SenderID:   m.SenderID,
			Type:       repository.MessageTypeContract,
			ContractID: m.ContractID,
			Status:     m.ContractStatus,
			Text:       m.Text,
			CreatedAt:  m.CreatedAt,
		}
	default:
		sys := m.IsSystem
		return models.ChatMessage{
			ID:        m.ID,
			ChatID:    chatID,
			SenderID:  m.SenderID,
			Type:      repository.MessageTypeText,
			Text:      m.Text,
			CreatedAt: m.CreatedAt,
			IsSystem:  &sys,
		}
	}
}

// Собираем ws-payload из сохраненной строки сообщения.
func WSMessagePayloadFromRow(chatID int, m repository.MessageRow) models.WSMessagePayload {
	cm := MessageRowToChatMessage(chatID, m)
	return models.WSMessagePayload{
		ID:         cm.ID,
		ChatID:     cm.ChatID,
		SenderID:   cm.SenderID,
		Type:       cm.Type,
		ContractID: cm.ContractID,
		Status:     cm.Status,
		Text:       cm.Text,
		CreatedAt:  cm.CreatedAt.UTC(),
		IsSystem:   cm.IsSystem,
	}
}

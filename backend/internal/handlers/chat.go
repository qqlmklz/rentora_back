package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"

	"rentora/backend/internal/middleware"
	"rentora/backend/internal/models"
	"rentora/backend/internal/repository"
	"rentora/backend/internal/services"
	"rentora/backend/internal/utils"

	"github.com/gin-gonic/gin"
)

// Обработчик POST /api/chats: создаем чат или берем уже существующий (buyer = текущий пользователь).
func CreateChat(chatService *services.ChatService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := middleware.GetUserID(c)
		if !ok {
			utils.JSONErrorUnauthorized(c, "Требуется авторизация")
			return
		}
		bodyBytes, err := io.ReadAll(c.Request.Body)
		if err != nil {
			log.Printf("[chats] CreateChat read body: %v", err)
			utils.JSONErrorBadRequest(c, "Не удалось прочитать тело запроса")
			return
		}
		log.Printf("[chats] CreateChat incoming body: %s", string(bodyBytes))

		var req models.CreateChatRequest
		if err := json.Unmarshal(bodyBytes, &req); err != nil {
			log.Printf("[chats] CreateChat JSON bind error: %v", err)
			utils.JSONErrorBadRequest(c, "Некорректный JSON")
			return
		}
		if req.PropertyID == nil {
			utils.JSONErrorBadRequest(c, "propertyId обязателен")
			return
		}
		if *req.PropertyID < 1 {
			utils.JSONErrorBadRequest(c, "propertyId должен быть положительным числом")
			return
		}
		propertyID := *req.PropertyID

		resp, err := chatService.CreateOrGetChat(c.Request.Context(), userID, propertyID)
		if err != nil {
			switch {
			case errors.Is(err, repository.ErrPropertyNotFound):
				utils.JSONErrorNotFound(c, "Объявление не найдено")
			case errors.Is(err, services.ErrChatSelf):
				utils.JSONErrorBadRequest(c, "Нельзя написать самому себе по своему объявлению")
			default:
				log.Printf("[chats] CreateChat: %v", err)
				utils.JSONErrorInternal(c, "Ошибка создания чата")
			}
			return
		}
		c.JSON(http.StatusOK, resp)
	}
}

// Обработчик GET /api/chats: отдаем список чатов текущего пользователя.
func ListChats(chatService *services.ChatService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := middleware.GetUserID(c)
		if !ok {
			utils.JSONErrorUnauthorized(c, "Требуется авторизация")
			return
		}
		items, err := chatService.ListMyChats(c.Request.Context(), userID)
		if err != nil {
			log.Printf("[chats] ListChats: %v", err)
			utils.JSONErrorInternal(c, "Ошибка загрузки чатов")
			return
		}
		if items == nil {
			items = []models.ChatListItem{}
		}
		c.JSON(http.StatusOK, items)
	}
}

// Обработчик GET /api/chats/:id: отдаем мету чата (объявление + собеседник).
func GetChat(chatService *services.ChatService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := middleware.GetUserID(c)
		if !ok {
			utils.JSONErrorUnauthorized(c, "Требуется авторизация")
			return
		}
		chatID, err := strconv.Atoi(c.Param("id"))
		if err != nil || chatID < 1 {
			utils.JSONErrorBadRequest(c, "Некорректный id чата")
			return
		}
		resp, err := chatService.GetChatDetail(c.Request.Context(), userID, chatID)
		if err != nil {
			if errors.Is(err, repository.ErrChatForbidden) {
				utils.JSONErrorForbidden(c, "Нет доступа к этому чату")
				return
			}
			log.Printf("[chats] GetChat: %v", err)
			utils.JSONErrorInternal(c, "Ошибка загрузки чата")
			return
		}
		c.JSON(http.StatusOK, resp)
	}
}

// Обработчик GET /api/chats/:id/messages: отдаем сообщения чата.
func GetChatMessages(chatService *services.ChatService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := middleware.GetUserID(c)
		if !ok {
			utils.JSONErrorUnauthorized(c, "Требуется авторизация")
			return
		}
		chatID, err := strconv.Atoi(c.Param("id"))
		if err != nil || chatID < 1 {
			utils.JSONErrorBadRequest(c, "Некорректный id чата")
			return
		}
		resp, err := chatService.ListMessages(c.Request.Context(), userID, chatID)
		if err != nil {
			if errors.Is(err, repository.ErrChatForbidden) {
				utils.JSONErrorForbidden(c, "Нет доступа к этому чату")
				return
			}
			log.Printf("[chats] GetChatMessages: %v", err)
			utils.JSONErrorInternal(c, "Ошибка загрузки сообщений")
			return
		}
		if resp.Messages == nil {
			resp.Messages = []models.ChatMessage{}
		}
		c.JSON(http.StatusOK, resp)
	}
}

// Обработчик PATCH /api/chats/:id/read: помечаем входящие сообщения как прочитанные.
func MarkChatRead(chatService *services.ChatService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := middleware.GetUserID(c)
		if !ok {
			utils.JSONErrorUnauthorized(c, "Требуется авторизация")
			return
		}
		chatID, err := strconv.Atoi(c.Param("id"))
		if err != nil || chatID < 1 {
			utils.JSONErrorBadRequest(c, "Некорректный id чата")
			return
		}
		if err := chatService.MarkChatRead(c.Request.Context(), userID, chatID); err != nil {
			if errors.Is(err, repository.ErrChatForbidden) {
				utils.JSONErrorForbidden(c, "Нет доступа к этому чату")
				return
			}
			log.Printf("[chats] MarkChatRead: %v", err)
			utils.JSONErrorInternal(c, "Ошибка обновления чата")
			return
		}
		c.Status(http.StatusNoContent)
	}
}

// Обработчик POST /api/chats/:id/messages: отправляем сообщение в чат.
func SendChatMessage(chatService *services.ChatService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := middleware.GetUserID(c)
		if !ok {
			utils.JSONErrorUnauthorized(c, "Требуется авторизация")
			return
		}
		chatID, err := strconv.Atoi(c.Param("id"))
		if err != nil || chatID < 1 {
			utils.JSONErrorBadRequest(c, "Некорректный id чата")
			return
		}
		var req models.SendMessageRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			utils.JSONErrorBadRequest(c, "Некорректное тело запроса")
			return
		}
		msg, err := chatService.SendMessage(c.Request.Context(), userID, chatID, req.Text)
		if err != nil {
			switch {
			case errors.Is(err, repository.ErrChatForbidden):
				utils.JSONErrorForbidden(c, "Нет доступа к этому чату")
			case errors.Is(err, services.ErrEmptyMessage):
				utils.JSONErrorBadRequest(c, "Текст сообщения не может быть пустым")
			case errors.Is(err, services.ErrMessageTooLong):
				utils.JSONErrorBadRequest(c, "Сообщение слишком длинное")
			default:
				log.Printf("[chats] SendChatMessage: %v", err)
				utils.JSONErrorInternal(c, "Ошибка отправки сообщения")
			}
			return
		}
		c.JSON(http.StatusCreated, msg)
	}
}

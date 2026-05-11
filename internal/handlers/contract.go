package handlers

import (
	"errors"
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

// Обработчик GET /api/chats/:chatId/contract-draft.
func GetContractDraft(contractService *services.ContractService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := middleware.GetUserID(c)
		if !ok {
			utils.JSONErrorUnauthorized(c, "Требуется авторизация")
			return
		}
		raw := c.Param("id")
		chatID, err := strconv.Atoi(raw)
		if err != nil {
			log.Printf("[contracts] contract-draft: invalid chatId=%q parse err=%v", raw, err)
			utils.JSONErrorBadRequest(c, "Некорректный id чата: ожидается положительное число")
			return
		}
		if chatID < 1 {
			log.Printf("[contracts] contract-draft: invalid chatId=%d (must be >= 1)", chatID)
			utils.JSONErrorBadRequest(c, "Некорректный id чата: ожидается положительное число")
			return
		}
		draft, err := contractService.GetContractDraft(c.Request.Context(), userID, chatID)
		if err != nil {
			switch {
			case errors.Is(err, repository.ErrChatNotFound):
				utils.JSONErrorNotFound(c, "Чат не найден")
			case errors.Is(err, repository.ErrChatForbidden):
				utils.JSONErrorForbidden(c, "Нет доступа к этому чату")
			case errors.Is(err, repository.ErrPropertyNotFound):
				utils.JSONErrorNotFound(c, "Объявление не найдено")
			case errors.Is(err, services.ErrContractDraftParticipantMissing):
				utils.JSONErrorNotFound(c, "Участник чата не найден")
			default:
				log.Printf("[contracts] GetContractDraft: %v", err)
				utils.JSONErrorInternal(c, "Ошибка загрузки черновика")
			}
			return
		}
		c.JSON(http.StatusOK, draft)
	}
}

// Обработчик POST /api/chats/:id/contracts.
func CreateChatContract(contractService *services.ContractService) gin.HandlerFunc {
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
		var req models.CreateContractBody
		if err := c.ShouldBindJSON(&req); err != nil {
			log.Printf("[contracts] CreateChatContract bind error: %v", err)
			utils.JSONErrorBadRequest(c, utils.FormatRequestBindError(err, &req))
			return
		}
		resp, err := contractService.CreateContractFromChat(c.Request.Context(), userID, chatID, req)
		if err != nil {
			switch {
			case errors.Is(err, repository.ErrChatNotFound):
				utils.JSONErrorNotFound(c, "Чат не найден")
			case errors.Is(err, repository.ErrContractForbidden):
				utils.JSONErrorForbidden(c, "Только арендодатель может создать договор")
			default:
				log.Printf("[contracts] CreateChatContract: %v", err)
				utils.JSONErrorInternal(c, "Ошибка создания договора")
			}
			return
		}
		c.JSON(http.StatusCreated, resp)
	}
}

// Обработчик GET /api/contracts/:id (id = contracts.id).
func GetContract(contractService *services.ContractService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := middleware.GetUserID(c)
		if !ok {
			utils.JSONErrorUnauthorized(c, "Требуется авторизация")
			return
		}
		raw := c.Param("id")
		contractID, err := strconv.Atoi(raw)
		if err != nil || contractID < 1 {
			log.Printf("[contracts] GET /api/contracts/:id invalid param raw=%q err=%v", raw, err)
			utils.JSONErrorBadRequest(c, "Некорректный id договора")
			return
		}
		log.Printf("[contracts] GET /api/contracts/:id requested contracts.id=%d userId=%d", contractID, userID)
		resp, err := contractService.GetContractByID(c.Request.Context(), userID, contractID)
		if err != nil {
			switch {
			case errors.Is(err, repository.ErrContractForbidden):
				utils.JSONErrorForbidden(c, "Нет доступа к договору")
			case errors.Is(err, repository.ErrContractNotFound):
				utils.JSONErrorNotFound(c, "Договор не найден")
			default:
				log.Printf("[contracts] GetContract: %v", err)
				utils.JSONErrorInternal(c, "Ошибка загрузки договора")
			}
			return
		}
		c.JSON(http.StatusOK, resp)
	}
}

// Обработчик PATCH /api/contracts/:id/accept.
func AcceptContract(contractService *services.ContractService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := middleware.GetUserID(c)
		if !ok {
			utils.JSONErrorUnauthorized(c, "Требуется авторизация")
			return
		}
		contractID, err := strconv.Atoi(c.Param("id"))
		if err != nil || contractID < 1 {
			utils.JSONErrorBadRequest(c, "Некорректный id договора")
			return
		}
		err = contractService.AcceptContract(c.Request.Context(), userID, contractID)
		if err != nil {
			switch {
			case errors.Is(err, repository.ErrContractForbidden):
				utils.JSONErrorForbidden(c, "Только арендатор может принять договор")
			case errors.Is(err, repository.ErrContractNotFound):
				utils.JSONErrorNotFound(c, "Договор не найден")
			case errors.Is(err, services.ErrContractWrongStatus):
				utils.JSONErrorBadRequest(c, "Договор нельзя принять в текущем статусе")
			default:
				log.Printf("[contracts] AcceptContract: %v", err)
				utils.JSONErrorInternal(c, "Ошибка обновления договора")
			}
			return
		}
		c.Status(http.StatusNoContent)
	}
}

// Обработчик PATCH /api/contracts/:id/reject.
func RejectContract(contractService *services.ContractService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := middleware.GetUserID(c)
		if !ok {
			utils.JSONErrorUnauthorized(c, "Требуется авторизация")
			return
		}
		contractID, err := strconv.Atoi(c.Param("id"))
		if err != nil || contractID < 1 {
			utils.JSONErrorBadRequest(c, "Некорректный id договора")
			return
		}
		err = contractService.RejectContract(c.Request.Context(), userID, contractID)
		if err != nil {
			switch {
			case errors.Is(err, repository.ErrContractForbidden):
				utils.JSONErrorForbidden(c, "Только арендатор может отклонить договор")
			case errors.Is(err, repository.ErrContractNotFound):
				utils.JSONErrorNotFound(c, "Договор не найден")
			case errors.Is(err, services.ErrContractWrongStatus):
				utils.JSONErrorBadRequest(c, "Договор нельзя отклонить в текущем статусе")
			default:
				log.Printf("[contracts] RejectContract: %v", err)
				utils.JSONErrorInternal(c, "Ошибка обновления договора")
			}
			return
		}
		c.Status(http.StatusNoContent)
	}
}

// Обработчик PATCH /api/contracts/:id/terminate.
// Тут :id — это id договора (в доках иногда пишут :contractId), но оставили :id для единообразия маршрутов в Gin.
func TerminateContract(contractService *services.ContractService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := middleware.GetUserID(c)
		if !ok {
			utils.JSONErrorUnauthorized(c, "Требуется авторизация")
			return
		}
		raw := c.Param("id")
		contractID, err := strconv.Atoi(raw)
		if err != nil || contractID < 1 {
			log.Printf("[contracts] terminate handler: contractIdFromURL=%q currentUserId=%d parseErr=%v reason=invalid_contract_id", raw, userID, err)
			utils.JSONErrorBadRequest(c, "Некорректный id договора в URL")
			return
		}
		log.Printf("[contracts] terminate handler: contractId=%d currentUserId=%d step=request", contractID, userID)

		err = contractService.TerminateContract(c.Request.Context(), userID, contractID)
		if err != nil {
			switch {
			case errors.Is(err, repository.ErrContractNotFound):
				log.Printf("[contracts] terminate handler: contractId=%d currentUserId=%d found=false reason=response_404", contractID, userID)
				utils.JSONErrorNotFound(c, "Договор не найден")
			case errors.Is(err, repository.ErrContractForbidden):
				log.Printf("[contracts] terminate handler: contractId=%d currentUserId=%d reason=response_403_forbidden", contractID, userID)
				utils.JSONErrorForbidden(c, "Нет доступа к договору")
			case errors.Is(err, services.ErrContractAlreadyTerminated):
				log.Printf("[contracts] terminate handler: contractId=%d currentUserId=%d reason=response_400_already_terminated", contractID, userID)
				utils.JSONErrorBadRequest(c, "Договор уже расторгнут")
			case errors.Is(err, services.ErrContractMustBeAcceptedToTerminate):
				log.Printf("[contracts] terminate handler: contractId=%d currentUserId=%d reason=response_400_not_accepted", contractID, userID)
				utils.JSONErrorBadRequest(c, "Можно расторгнуть только принятый договор")
			case errors.Is(err, services.ErrContractMissingChat):
				log.Printf("[contracts] terminate handler: contractId=%d currentUserId=%d reason=response_400_no_chat", contractID, userID)
				utils.JSONErrorBadRequest(c, "У договора не указан чат")
			default:
				log.Printf("[contracts] terminate handler: contractId=%d currentUserId=%d reason=response_500 err=%v", contractID, userID, err)
				utils.JSONErrorInternal(c, err.Error())
			}
			return
		}

		log.Printf("[contracts] terminate handler: contractId=%d currentUserId=%d ok response=200 status=terminated", contractID, userID)
		c.JSON(http.StatusOK, gin.H{
			"message": "Договор расторгнут",
			"status":  "terminated",
		})
	}
}

// Обработчик GET /api/profile/documents.
func GetProfileDocuments(contractService *services.ContractService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := middleware.GetUserID(c)
		if !ok {
			utils.JSONErrorUnauthorized(c, "Требуется авторизация")
			return
		}
		items, err := contractService.ListAcceptedDocuments(c.Request.Context(), userID)
		if err != nil {
			log.Printf("[contracts] GetProfileDocuments: %v", err)
			utils.JSONErrorInternal(c, "Ошибка загрузки документов")
			return
		}
		if items == nil {
			items = []models.ContractDocumentItem{}
		}
		c.JSON(http.StatusOK, items)
	}
}

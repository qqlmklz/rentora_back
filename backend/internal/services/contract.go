package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"rentora/backend/internal/models"
	"rentora/backend/internal/repository"
	"rentora/backend/internal/ws"
)

// Сервис работы с договорами аренды внутри чатов.
type ContractService struct {
	repo *repository.DB
	hub  *ws.Hub
}

// Конструктор ContractService.
func NewContractService(repo *repository.DB, hub *ws.Hub) *ContractService {
	return &ContractService{repo: repo, hub: hub}
}

const (
	msgContractSubmitted  = "Договор отправлен на согласование"
	msgContractAccepted   = "Договор принят"
	msgContractRejected   = "Договор не принят"
	msgContractTerminated = "Договор больше не действителен"
)

// Ошибка, когда accept/reject нельзя сделать в текущем статусе.
var ErrContractWrongStatus = errors.New("contract wrong status")

// Ошибка, если пытаемся расторгнуть договор, который уже расторгнут.
var ErrContractAlreadyTerminated = errors.New("contract already terminated")

// Ошибка, если расторжение вызвали не для статуса accepted.
var ErrContractMustBeAcceptedToTerminate = errors.New("contract must be accepted to terminate")

// Ошибка, если у договора нет корректного contracts.chat_id.
var ErrContractMissingChat = errors.New("contract has no valid chat_id")

// Ошибка, если не нашли пользователя арендодателя или арендатора для черновика.
var ErrContractDraftParticipantMissing = errors.New("contract draft: landlord or tenant not found")

// Собираем текст договора из полей формы.
func BuildContractText(in models.ContractFormData) string {
	date := strings.TrimSpace(in.ContractDate)
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Договор аренды недвижимости\n\n")
	fmt.Fprintf(&b, "Дата составления: %s\n", date)
	fmt.Fprintf(&b, "Арендодатель: %s\n", strings.TrimSpace(in.LandlordName))
	fmt.Fprintf(&b, "Арендатор: %s\n", strings.TrimSpace(in.TenantName))
	if strings.TrimSpace(in.Title) != "" {
		fmt.Fprintf(&b, "Объект: %s\n", strings.TrimSpace(in.Title))
	}
	fmt.Fprintf(&b, "Город: %s\n", strings.TrimSpace(in.City))
	fmt.Fprintf(&b, "Район: %s\n", strings.TrimSpace(in.District))
	fmt.Fprintf(&b, "Адрес: %s\n", strings.TrimSpace(in.Address))
	fmt.Fprintf(&b, "Тип аренды: %s\n", strings.TrimSpace(in.RentType))
	fmt.Fprintf(&b, "Тип жилья: %s\n", strings.TrimSpace(in.PropertyType))
	fmt.Fprintf(&b, "Арендная плата: %d руб.\n", in.Price)
	if in.Deposit != nil {
		fmt.Fprintf(&b, "Залог: %d руб.\n", *in.Deposit)
	} else {
		fmt.Fprintf(&b, "Залог: не указан\n")
	}
	fmt.Fprintf(&b, "Коммунальные включены в стоимость: %s\n", boolRu(in.UtilitiesIncluded))
	if in.UtilitiesPrice != nil {
		fmt.Fprintf(&b, "Коммунальные платежи (сумма): %d руб.\n", *in.UtilitiesPrice)
	}
	if strings.TrimSpace(in.Prepayment) != "" {
		fmt.Fprintf(&b, "Предоплата: %s\n", strings.TrimSpace(in.Prepayment))
	}
	fmt.Fprintf(&b, "Дети: %s\n", boolRu(in.ChildrenAllowed))
	fmt.Fprintf(&b, "Животные: %s\n", boolRu(in.PetsAllowed))
	fmt.Fprintf(&b, "Дата начала: %s\n", strings.TrimSpace(in.StartDate))
	fmt.Fprintf(&b, "Дата окончания: %s\n", strings.TrimSpace(in.EndDate))
	return b.String()
}

func boolRu(v bool) string {
	if v {
		return "да"
	}
	return "нет"
}

// Собираем дефолтный черновик формы из объявления и имен участников.
func (s *ContractService) GetContractDraft(ctx context.Context, userID, chatID int) (*models.ContractDraftResponse, error) {
	log.Printf("[contracts] contract-draft: chatId=%d userId=%d step=start", chatID, userID)
	propID, landlordID, tenantID, err := s.repo.GetChatParticipantRoles(ctx, chatID)
	if err != nil {
		if errors.Is(err, repository.ErrChatNotFound) {
			log.Printf("[contracts] contract-draft: chatId=%d chatFound=false", chatID)
		} else {
			log.Printf("[contracts] contract-draft: chatId=%d chat lookup err=%v", chatID, err)
		}
		return nil, err
	}
	log.Printf("[contracts] contract-draft: chatId=%d chatFound=true propertyId=%d landlordId=%d tenantId=%d", chatID, propID, landlordID, tenantID)
	if userID != landlordID && userID != tenantID {
		log.Printf("[contracts] contract-draft: chatId=%d forbidden userId=%d (not participant)", chatID, userID)
		return nil, repository.ErrChatForbidden
	}
	pd, err := s.repo.GetPropertyByID(ctx, propID)
	if err != nil {
		if errors.Is(err, repository.ErrPropertyNotFound) {
			log.Printf("[contracts] contract-draft: chatId=%d propertyId=%d propertyFound=false", chatID, propID)
		} else {
			log.Printf("[contracts] contract-draft: chatId=%d propertyId=%d property err=%v", chatID, propID, err)
		}
		return nil, err
	}
	log.Printf("[contracts] contract-draft: chatId=%d propertyId=%d propertyFound=true", chatID, propID)
	landlord, err := s.repo.GetUserByID(ctx, landlordID)
	if err != nil {
		log.Printf("[contracts] contract-draft: chatId=%d landlordId=%d load err=%v", chatID, landlordID, err)
		return nil, err
	}
	if landlord == nil {
		log.Printf("[contracts] contract-draft: chatId=%d landlordId=%d landlordFound=false", chatID, landlordID)
		return nil, ErrContractDraftParticipantMissing
	}
	log.Printf("[contracts] contract-draft: chatId=%d landlordId=%d landlordFound=true", chatID, landlordID)
	tenant, err := s.repo.GetUserByID(ctx, tenantID)
	if err != nil {
		log.Printf("[contracts] contract-draft: chatId=%d tenantId=%d load err=%v", chatID, tenantID, err)
		return nil, err
	}
	if tenant == nil {
		log.Printf("[contracts] contract-draft: chatId=%d tenantId=%d tenantFound=false", chatID, tenantID)
		return nil, ErrContractDraftParticipantMissing
	}
	log.Printf("[contracts] contract-draft: chatId=%d tenantId=%d tenantFound=true ok", chatID, tenantID)
	prep := ""
	if pd.Prepayment != nil {
		prep = *pd.Prepayment
	}
	return &models.ContractDraftResponse{
		LandlordName:      landlord.Name,
		TenantName:        tenant.Name,
		City:              pd.City,
		Address:           pd.Address,
		District:          pd.District,
		RentType:          pd.RentType,
		PropertyType:      pd.PropertyType,
		Price:             pd.Price,
		Deposit:           pd.Deposit,
		UtilitiesIncluded: pd.UtilitiesIncluded,
		UtilitiesPrice:    pd.UtilitiesPrice,
		Prepayment:        prep,
		ChildrenAllowed:   pd.ChildrenAllowed,
		PetsAllowed:       pd.PetsAllowed,
		StartDate:         "",
		EndDate:           "",
		Title:             pd.Title,
	}, nil
}

// Создаем договор со статусом pending и системное сообщение типа contract (только арендодатель).
func (s *ContractService) CreateContractFromChat(ctx context.Context, userID, chatID int, in models.CreateContractBody) (*models.PostChatContractResponse, error) {
	propID, landlordID, tenantID, err := s.repo.GetChatParticipantRoles(ctx, chatID)
	if err != nil {
		return nil, err
	}
	if userID != landlordID {
		return nil, repository.ErrContractForbidden
	}
	data := models.CreateContractBodyToFormData(in)
	if strings.TrimSpace(data.ContractDate) == "" {
		data.ContractDate = time.Now().Format("2006-01-02")
	}
	if inJSON, err := json.Marshal(in); err == nil {
		log.Printf("[contracts] create: chatId=%d input JSON: %s", chatID, string(inJSON))
	}
	raw, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	log.Printf("[contracts] create: chatId=%d contractData bytes=%d payload=%s", chatID, len(raw), truncateForLog(string(raw), 4000))
	text := BuildContractText(data)
	log.Printf("[contracts] create: chatId=%d contractText before save len=%d preview=%s", chatID, len(text), truncateForLog(text, 500))
	cid, m, err := s.repo.CreatePendingContractWithSystemMessage(ctx, chatID, propID, landlordID, tenantID, raw, text, msgContractSubmitted)
	if err != nil {
		return nil, err
	}
	log.Printf("[contracts] create: inserted contracts.id=%d message.id=%d message.contract_id=%v", cid, m.ID, m.ContractID)
	s.broadcastMessage(ctx, chatID, m)
	row, err := s.repo.GetContractByID(ctx, cid)
	if err != nil {
		return nil, err
	}
	log.Printf("[contracts] create: contractId=%d contractText after read from DB len=%d preview=%s", cid, len(row.ContractText), truncateForLog(row.ContractText, 500))
	cr := contractRowToResponse(row)
	apiMsg := MessageRowToChatMessage(chatID, m)
	log.Printf("[contracts] create: response message.contractId (API)=%v", apiMsg.ContractID)
	return &models.PostChatContractResponse{
		ID:           cr.ID,
		ContractID:   cr.ID,
		Status:       cr.Status,
		ContractData: cr.ContractData,
		ContractText: cr.ContractText,
		Contract:     *cr,
		Message:      apiMsg,
	}, nil
}

// Возвращаем полный договор, если пользователь арендодатель или арендатор.
func (s *ContractService) GetContractByID(ctx context.Context, userID, contractID int) (*models.ContractResponse, error) {
	row, err := s.repo.GetContractByID(ctx, contractID)
	if err != nil {
		return nil, err
	}
	ok, err := s.repo.IsChatParticipant(ctx, row.ChatID, userID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, repository.ErrContractForbidden
	}
	log.Printf("[contracts] get: contractId=%d contractText from DB len=%d preview=%s", contractID, len(row.ContractText), truncateForLog(row.ContractText, 500))
	return contractRowToResponse(row), nil
}

// Ставим статус accepted и добавляем системное сообщение (только арендатор).
func (s *ContractService) AcceptContract(ctx context.Context, userID, contractID int) error {
	row, err := s.repo.GetContractByID(ctx, contractID)
	if err != nil {
		return err
	}
	log.Printf("[contracts] accept: contractId=%d tenantId=%d currentUserId=%d currentStatus=%s propertyId=%d", contractID, row.TenantID, userID, row.Status, row.PropertyID)
	if userID != row.TenantID {
		return repository.ErrContractForbidden
	}
	if row.Status != repository.ContractStatusPending {
		return ErrContractWrongStatus
	}
	m, err := s.repo.AcceptContractWithMessage(ctx, contractID, userID, row.ChatID, msgContractAccepted)
	if err != nil {
		return err
	}
	log.Printf("[contracts] accept: contractId=%d newStatus=%s propertyId=%d", contractID, repository.ContractStatusAccepted, row.PropertyID)
	s.broadcastMessage(ctx, row.ChatID, m)
	return nil
}

// Ставим статус rejected и добавляем системное сообщение (только арендатор).
func (s *ContractService) RejectContract(ctx context.Context, userID, contractID int) error {
	row, err := s.repo.GetContractByID(ctx, contractID)
	if err != nil {
		return err
	}
	if userID != row.TenantID {
		return repository.ErrContractForbidden
	}
	if row.Status != repository.ContractStatusPending {
		return ErrContractWrongStatus
	}
	m, err := s.repo.RejectContractWithMessage(ctx, contractID, userID, row.ChatID, msgContractRejected)
	if err != nil {
		return err
	}
	s.broadcastMessage(ctx, row.ChatID, m)
	return nil
}

// Ставим статус terminated и добавляем системное сообщение (арендодатель или арендатор, только из accepted).
func (s *ContractService) TerminateContract(ctx context.Context, userID, contractID int) error {
	log.Printf("[contracts] terminate: contractId=%d currentUserId=%d step=load", contractID, userID)
	row, err := s.repo.GetContractByID(ctx, contractID)
	if err != nil {
		if errors.Is(err, repository.ErrContractNotFound) {
			log.Printf("[contracts] terminate: contractId=%d currentUserId=%d found=false reason=not_found", contractID, userID)
		} else {
			log.Printf("[contracts] terminate: contractId=%d currentUserId=%d found=unknown reason=load_error err=%v", contractID, userID, err)
		}
		return err
	}
	log.Printf("[contracts] terminate: contractId=%d currentUserId=%d found=true chatId=%d currentStatus=%s", contractID, userID, row.ChatID, row.Status)
	if row.ChatID < 1 {
		log.Printf("[contracts] terminate: contractId=%d currentUserId=%d chatId=%d reason=missing_or_invalid_chat", contractID, userID, row.ChatID)
		return ErrContractMissingChat
	}
	if userID != row.LandlordID && userID != row.TenantID {
		log.Printf("[contracts] terminate: contractId=%d currentUserId=%d currentStatus=%s reason=forbidden user_not_participant landlordId=%d tenantId=%d",
			contractID, userID, row.Status, row.LandlordID, row.TenantID)
		return repository.ErrContractForbidden
	}
	switch row.Status {
	case repository.ContractStatusTerminated:
		log.Printf("[contracts] terminate: contractId=%d currentUserId=%d currentStatus=%s reason=reject not_accepted_already_terminated", contractID, userID, row.Status)
		return ErrContractAlreadyTerminated
	case repository.ContractStatusAccepted:
		// статус подходит, идем дальше
	default:
		log.Printf("[contracts] terminate: contractId=%d currentUserId=%d currentStatus=%s reason=reject not_accepted", contractID, userID, row.Status)
		return ErrContractMustBeAcceptedToTerminate
	}
	m, err := s.repo.TerminateContractWithMessage(ctx, contractID, row.ChatID, userID, msgContractTerminated)
	if err != nil {
		log.Printf("[contracts] terminate: contractId=%d currentUserId=%d chatId=%d currentStatus=accepted reason=persist_failed err=%v", contractID, userID, row.ChatID, err)
		return err
	}
	log.Printf("[contracts] terminate: contractId=%d currentUserId=%d ok oldStatus=accepted newStatus=terminated chatId=%d systemMessageId=%d",
		contractID, userID, row.ChatID, m.ID)
	s.broadcastMessage(ctx, row.ChatID, m)
	return nil
}

// Возвращаем для профиля договоры со статусами accepted и terminated (история документов).
func (s *ContractService) ListAcceptedDocuments(ctx context.Context, userID int) ([]models.ContractDocumentItem, error) {
	rows, err := s.repo.ListContractsForProfileDocuments(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]models.ContractDocumentItem, 0, len(rows))
	for _, r := range rows {
		out = append(out, models.ContractDocumentItem{
			ID:            r.ID,
			ChatID:        r.ChatID,
			PropertyID:    r.PropertyID,
			PropertyTitle: r.PropertyTitle,
			LandlordID:    r.LandlordID,
			TenantID:      r.TenantID,
			Status:        r.Status,
			ContractData:  jsonRaw(r.ContractData),
			ContractText:  contractTextFromStored(r.ContractData, r.ContractText, r.ID),
			CreatedAt:     r.CreatedAt,
			UpdatedAt:     r.UpdatedAt,
		})
	}
	return out, nil
}

func jsonRaw(b []byte) json.RawMessage {
	if len(b) == 0 {
		return json.RawMessage(`{}`)
	}
	return json.RawMessage(b)
}

func contractRowToResponse(r *repository.ContractRow) *models.ContractResponse {
	text := contractTextFromStored(r.ContractData, r.ContractText, r.ID)
	return &models.ContractResponse{
		ID:           r.ID,
		ChatID:       r.ChatID,
		PropertyID:   r.PropertyID,
		LandlordID:   r.LandlordID,
		TenantID:     r.TenantID,
		Status:       r.Status,
		ContractData: jsonRaw(r.ContractData),
		ContractText: text,
		CreatedAt:    r.CreatedAt,
		UpdatedAt:    r.UpdatedAt,
	}
}

// Берем сохраненный текст договора, а если пусто — пересобираем из contract_data JSON.
func contractTextFromStored(contractData []byte, stored string, contractID int) string {
	text := strings.TrimSpace(stored)
	if text != "" {
		return text
	}
	if len(bytes.TrimSpace(contractData)) == 0 {
		log.Printf("[contracts] contract id=%d: contractText empty, contract_data empty", contractID)
		return ""
	}
	var fd models.ContractFormData
	if err := json.Unmarshal(contractData, &fd); err != nil {
		log.Printf("[contracts] contract id=%d: unmarshal contract_data: %v", contractID, err)
		return ""
	}
	rebuilt := BuildContractText(fd)
	log.Printf("[contracts] contract id=%d: rebuilt contractText from contractData (len=%d)", contractID, len(rebuilt))
	return rebuilt
}

func truncateForLog(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}

func (s *ContractService) broadcastMessage(ctx context.Context, chatID int, m repository.MessageRow) {
	if s.hub == nil {
		return
	}
	sellerID, buyerID, err := s.repo.GetChatParticipantIDs(ctx, chatID)
	if err != nil {
		log.Printf("[contracts] GetChatParticipantIDs for ws: %v", err)
		return
	}
	s.hub.BroadcastNewMessage(chatID, sellerID, buyerID, WSMessagePayloadFromRow(chatID, m))
}

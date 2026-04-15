package models

import (
	"encoding/json"
	"time"
)

// Ответ для GET /api/chats/:chatId/contract-draft — только поля формы договора.
// Поле Title — это название объявления (нужно при POST /api/chats/:id/contracts).
type ContractDraftResponse struct {
	LandlordName      string `json:"landlordName"`
	TenantName        string `json:"tenantName"`
	City              string `json:"city"`
	Address           string `json:"address"`
	District          string `json:"district"`
	RentType          string `json:"rentType"`
	PropertyType      string `json:"propertyType"`
	Price             int    `json:"price"`
	Deposit           *int   `json:"deposit,omitempty"`
	UtilitiesIncluded bool   `json:"utilitiesIncluded"`
	UtilitiesPrice    *int   `json:"utilitiesPrice,omitempty"`
	Prepayment        string `json:"prepayment,omitempty"`
	ChildrenAllowed   bool   `json:"childrenAllowed"`
	PetsAllowed       bool   `json:"petsAllowed"`
	StartDate         string `json:"startDate"`
	EndDate           string `json:"endDate"`
	Title             string `json:"title"` // берем из объявления; нужно для тела POST /contracts
}

// Это то, что храним в contract_data (JSON), и из этого же собираем текст через BuildContractText.
type ContractFormData struct {
	Title             string `json:"title,omitempty"`
	City              string `json:"city"`
	Address           string `json:"address"`
	District          string `json:"district"`
	RentType          string `json:"rentType"`
	PropertyType      string `json:"propertyType"`
	Price             int    `json:"price"`
	Deposit           *int   `json:"deposit,omitempty"`
	UtilitiesIncluded bool   `json:"utilitiesIncluded"`
	UtilitiesPrice    *int   `json:"utilitiesPrice,omitempty"`
	Prepayment        string `json:"prepayment,omitempty"`
	ChildrenAllowed   bool   `json:"childrenAllowed"`
	PetsAllowed       bool   `json:"petsAllowed"`
	StartDate         string `json:"startDate"`
	EndDate           string `json:"endDate"`
	LandlordName      string `json:"landlordName"`
	TenantName        string `json:"tenantName"`
	ContractDate      string `json:"contractDate,omitempty"`
}

// Тело JSON для POST /api/chats/:chatId/contracts.
type CreateContractBody struct {
	LandlordName      string `json:"landlordName" binding:"required"`
	TenantName        string `json:"tenantName" binding:"required"`
	City              string `json:"city" binding:"required"`
	ContractDate      string `json:"contractDate" binding:"required"`
	Address           string `json:"address" binding:"required"`
	District          string `json:"district" binding:"required"`
	RentType          string `json:"rentType" binding:"required"`
	PropertyType      string `json:"propertyType" binding:"required"`
	Price             int    `json:"price" binding:"required"`
	Deposit           *int   `json:"deposit,omitempty"`
	UtilitiesIncluded bool   `json:"utilitiesIncluded"`
	UtilitiesPrice    *int   `json:"utilitiesPrice,omitempty"`
	Prepayment        string `json:"prepayment,omitempty"`
	ChildrenAllowed   bool   `json:"childrenAllowed"`
	PetsAllowed       bool   `json:"petsAllowed"`
	StartDate         string `json:"startDate" binding:"required"`
	EndDate           string `json:"endDate" binding:"required"`
}

// Перекладываем API-тело в формат для хранения (title пустой, строку «Объект» в тексте не печатаем).
func CreateContractBodyToFormData(in CreateContractBody) ContractFormData {
	return ContractFormData{
		Title:             "",
		LandlordName:      in.LandlordName,
		TenantName:        in.TenantName,
		City:              in.City,
		ContractDate:      in.ContractDate,
		Address:           in.Address,
		District:          in.District,
		RentType:          in.RentType,
		PropertyType:      in.PropertyType,
		Price:             in.Price,
		Deposit:           in.Deposit,
		UtilitiesIncluded: in.UtilitiesIncluded,
		UtilitiesPrice:    in.UtilitiesPrice,
		Prepayment:        in.Prepayment,
		ChildrenAllowed:   in.ChildrenAllowed,
		PetsAllowed:       in.PetsAllowed,
		StartDate:         in.StartDate,
		EndDate:           in.EndDate,
	}
}

// Успешный ответ для POST /api/chats/:chatId/contracts.
// Поля id и contractId — это один и тот же PK из contracts.id; просто дублируем сверху и внутри contract.
type PostChatContractResponse struct {
	ID           int             `json:"id"`
	ContractID   int             `json:"contractId"`
	Status       string          `json:"status"`
	ContractData json.RawMessage `json:"contractData"`
	ContractText string          `json:"contractText"`
	Contract     ContractResponse `json:"contract"`
	Message      ChatMessage      `json:"message"`
}

// Полный ответ для GET /api/contracts/:id.
type ContractResponse struct {
	ID           int             `json:"id"`
	ChatID       int             `json:"chatId"`
	PropertyID   int             `json:"propertyId"`
	LandlordID   int             `json:"landlordId"`
	TenantID     int             `json:"tenantId"`
	// Возможные статусы: pending | accepted | rejected | terminated (иногда еще draft).
	Status       string          `json:"status"`
	ContractData json.RawMessage `json:"contractData"`
	ContractText string          `json:"contractText"`
	CreatedAt    time.Time       `json:"createdAt"`
	UpdatedAt    time.Time       `json:"updatedAt"`
}

// Одна запись для GET /api/profile/documents (берем accepted и terminated).
type ContractDocumentItem struct {
	ID            int             `json:"id"`
	ChatID        int             `json:"chatId"`
	PropertyID    int             `json:"propertyId"`
	PropertyTitle string          `json:"propertyTitle"`
	LandlordID    int             `json:"landlordId"`
	TenantID      int             `json:"tenantId"`
	Status        string          `json:"status"` // тут только accepted | terminated
	ContractData  json.RawMessage `json:"contractData"`
	ContractText  string          `json:"contractText"`
	CreatedAt     time.Time       `json:"createdAt"`
	UpdatedAt     time.Time       `json:"updatedAt"`
}

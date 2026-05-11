package models

// Валидированный вход для создания объявления.
type CreatePropertyInput struct {
	RentType        string
	Category        string
	PropertyType    string
	Title           string
	City            string
	District        string
	Price           int
	UtilitiesIncluded bool
	UtilitiesPrice  *int
	Deposit         *int
	CommissionPercent *int
	Prepayment      *string
	ChildrenAllowed bool
	PetsAllowed     bool
	Address         string
	Metro           *string
	ApartmentNumber *string
	Rooms           int
	TotalArea       float64
	LivingArea      *float64
	KitchenArea     *float64
	Floor           *int
	TotalFloors     *int
	HousingType     *string
}

// Ответ после успешного создания объявления.
type PropertyCreateResponse struct {
	ID       int      `json:"id"`
	Title    string   `json:"title"`
	City     string   `json:"city"`
	District string   `json:"district"`
	Images   []string `json:"images"`
}


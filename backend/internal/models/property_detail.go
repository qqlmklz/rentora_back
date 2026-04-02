package models

// PropertyDetail is the response for GET /api/properties/:id.
// apartmentNumber is only included for the owner (when JWT matches owner).
type PropertyDetail struct {
	ID                  int      `json:"id"`
	Title               string   `json:"title"`
	Price               int      `json:"price"`
	Category            string   `json:"category"`
	PropertyType        string   `json:"propertyType"`
	Rooms               int      `json:"rooms"`
	TotalArea           float64  `json:"totalArea"`
	LivingArea          *float64 `json:"livingArea,omitempty"`
	KitchenArea         *float64 `json:"kitchenArea,omitempty"`
	Floor               *int     `json:"floor,omitempty"`
	TotalFloors         *int     `json:"totalFloors,omitempty"`
	HousingType         *string  `json:"housingType,omitempty"`
	RentType            string   `json:"rentType"`
	Address             string   `json:"address"`
	City                string   `json:"city"`
	District            string   `json:"district"`
	ApartmentNumber     *string  `json:"apartmentNumber,omitempty"`
	Metro               *string  `json:"metro,omitempty"`
	UtilitiesIncluded   bool     `json:"utilitiesIncluded"`
	UtilitiesPrice      *int     `json:"utilitiesPrice,omitempty"`
	Deposit             *int     `json:"deposit,omitempty"`
	CommissionPercent   *int     `json:"commissionPercent,omitempty"`
	Prepayment          *string  `json:"prepayment,omitempty"`
	ChildrenAllowed     bool     `json:"childrenAllowed"`
	PetsAllowed         bool     `json:"petsAllowed"`
	Photos              []string `json:"photos"`
	OwnerID             *int     `json:"ownerId,omitempty"`
	OwnerName           *string  `json:"ownerName,omitempty"`
	OwnerAvatar         *string  `json:"ownerAvatar,omitempty"`
}

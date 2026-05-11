package models

// Тело JSON для PATCH /api/properties/:id (все поля опциональные).
type UpdatePropertyPatch struct {
	RentType            *string  `json:"rentType,omitempty"`
	Category            *string  `json:"category,omitempty"`
	PropertyType        *string  `json:"propertyType,omitempty"`
	Title               *string  `json:"title,omitempty"`
	City                *string  `json:"city,omitempty"`
	District            *string  `json:"district,omitempty"`
	Price               *int     `json:"price,omitempty"`
	UtilitiesIncluded   *bool    `json:"utilitiesIncluded,omitempty"`
	UtilitiesPrice      *int     `json:"utilitiesPrice,omitempty"`
	Deposit             *int     `json:"deposit,omitempty"`
	CommissionPercent   *int     `json:"commissionPercent,omitempty"`
	Prepayment          *string  `json:"prepayment,omitempty"`
	ChildrenAllowed     *bool    `json:"childrenAllowed,omitempty"`
	PetsAllowed         *bool    `json:"petsAllowed,omitempty"`
	Address             *string  `json:"address,omitempty"`
	Metro               *string  `json:"metro,omitempty"`
	ApartmentNumber     *string  `json:"apartmentNumber,omitempty"`
	Rooms               *int     `json:"rooms,omitempty"`
	TotalArea           *float64 `json:"totalArea,omitempty"`
	LivingArea          *float64 `json:"livingArea,omitempty"`
	KitchenArea         *float64 `json:"kitchenArea,omitempty"`
	Floor               *int     `json:"floor,omitempty"`
	TotalFloors         *int     `json:"totalFloors,omitempty"`
	HousingType         *string  `json:"housingType,omitempty"`
}

// Тело PATCH /api/properties/:id (и еще можно передать как поле "payload" в multipart).
// Поле ExistingPhotos: если задано (даже []), это список URL фото, которые оставляем в БД.
// Если значение nil, значит старые фото не трогаем, а только добавляем новые из multipart (если они есть).
// В multipart лучше передавать form-поле existingPhotos (JSON-строка массива); если оно непустое, кладем его в ExistingPhotos.
type UpdatePropertyPayload struct {
	UpdatePropertyPatch
	ExistingPhotos *[]string `json:"existingPhotos,omitempty"`
}

// Возвращает true, если есть обновления полей или явная синхронизация списка фото.
func (p UpdatePropertyPayload) HasMetaChanges() bool {
	return !p.UpdatePropertyPatch.IsEmpty() || p.ExistingPhotos != nil
}

// Возвращает true, если ни одно поле не задано.
func (p UpdatePropertyPatch) IsEmpty() bool {
	return p.RentType == nil && p.Category == nil && p.PropertyType == nil && p.Title == nil &&
		p.City == nil && p.District == nil && p.Price == nil && p.UtilitiesIncluded == nil &&
		p.UtilitiesPrice == nil && p.Deposit == nil && p.CommissionPercent == nil && p.Prepayment == nil &&
		p.ChildrenAllowed == nil && p.PetsAllowed == nil && p.Address == nil && p.Metro == nil &&
		p.ApartmentNumber == nil && p.Rooms == nil && p.TotalArea == nil && p.LivingArea == nil &&
		p.KitchenArea == nil && p.Floor == nil && p.TotalFloors == nil && p.HousingType == nil
}

// Применяем patch к base: обновляем только те поля, которые не nil.
func ApplyPropertyPatch(base *CreatePropertyInput, patch UpdatePropertyPatch) {
	if patch.RentType != nil {
		base.RentType = *patch.RentType
	}
	if patch.Category != nil {
		base.Category = *patch.Category
	}
	if patch.PropertyType != nil {
		base.PropertyType = *patch.PropertyType
	}
	if patch.Title != nil {
		base.Title = *patch.Title
	}
	if patch.City != nil {
		base.City = *patch.City
	}
	if patch.District != nil {
		base.District = *patch.District
	}
	if patch.Price != nil {
		base.Price = *patch.Price
	}
	if patch.UtilitiesIncluded != nil {
		base.UtilitiesIncluded = *patch.UtilitiesIncluded
	}
	if patch.UtilitiesPrice != nil {
		base.UtilitiesPrice = patch.UtilitiesPrice
	}
	if patch.Deposit != nil {
		base.Deposit = patch.Deposit
	}
	if patch.CommissionPercent != nil {
		base.CommissionPercent = patch.CommissionPercent
	}
	if patch.Prepayment != nil {
		base.Prepayment = patch.Prepayment
	}
	if patch.ChildrenAllowed != nil {
		base.ChildrenAllowed = *patch.ChildrenAllowed
	}
	if patch.PetsAllowed != nil {
		base.PetsAllowed = *patch.PetsAllowed
	}
	if patch.Address != nil {
		base.Address = *patch.Address
	}
	if patch.Metro != nil {
		base.Metro = patch.Metro
	}
	if patch.ApartmentNumber != nil {
		base.ApartmentNumber = patch.ApartmentNumber
	}
	if patch.Rooms != nil {
		base.Rooms = *patch.Rooms
	}
	if patch.TotalArea != nil {
		base.TotalArea = *patch.TotalArea
	}
	if patch.LivingArea != nil {
		base.LivingArea = patch.LivingArea
	}
	if patch.KitchenArea != nil {
		base.KitchenArea = patch.KitchenArea
	}
	if patch.Floor != nil {
		base.Floor = patch.Floor
	}
	if patch.TotalFloors != nil {
		base.TotalFloors = patch.TotalFloors
	}
	if patch.HousingType != nil {
		base.HousingType = patch.HousingType
	}
}

package models

import "strings"

// Коды подкатегории / propertyType в API (англ.).
const (
	PropertySubcategoryApartment = "apartment"
	PropertySubcategoryRoom      = "room"
	PropertySubcategoryHouse     = "house"
	PropertySubcategoryStudio    = "studio"
	PropertySubcategoryCottage   = "cottage"
	PropertySubcategoryOffice    = "office"
	PropertySubcategoryCoworking = "coworking"
	PropertySubcategoryBuilding  = "building"
	PropertySubcategoryWarehouse = "warehouse"
)

// Значения property_type в БД (рус.).
const (
	PropertyTypeStoredApartment = "квартира"
	PropertyTypeStoredRoom      = "комната"
	PropertyTypeStoredHouse     = "дом/дача"
	PropertyTypeStoredStudio    = "студия"
	PropertyTypeStoredCottage   = "коттедж"
	PropertyTypeStoredOffice    = "офис"
	PropertyTypeStoredCoworking = "коворкинг"
	PropertyTypeStoredBuilding  = "здание"
	PropertyTypeStoredWarehouse = "склад"
)

const (
	CategoryStoredResidential = "жилая"
	CategoryStoredCommercial  = "коммерческая"
)

// NormalizeCategoryStored приводит category к значению в БД.
func NormalizeCategoryStored(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "residential", "жилая":
		return CategoryStoredResidential
	case "commercial", "коммерческая":
		return CategoryStoredCommercial
	default:
		return strings.TrimSpace(v)
	}
}

// NormalizePropertySubcategoryAPI переводит subcategory из формы создания в property_type для БД.
func NormalizePropertySubcategoryAPI(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case PropertySubcategoryApartment, PropertyTypeStoredApartment:
		return PropertyTypeStoredApartment
	case PropertySubcategoryRoom, PropertyTypeStoredRoom:
		return PropertyTypeStoredRoom
	case PropertySubcategoryHouse, PropertyTypeStoredHouse:
		return PropertyTypeStoredHouse
	case PropertySubcategoryStudio, PropertyTypeStoredStudio:
		return PropertyTypeStoredStudio
	case PropertySubcategoryCottage, PropertyTypeStoredCottage:
		return PropertyTypeStoredCottage
	case PropertySubcategoryOffice, PropertyTypeStoredOffice:
		return PropertyTypeStoredOffice
	case PropertySubcategoryCoworking, PropertyTypeStoredCoworking:
		return PropertyTypeStoredCoworking
	case PropertySubcategoryBuilding, PropertyTypeStoredBuilding:
		return PropertyTypeStoredBuilding
	case PropertySubcategoryWarehouse, PropertyTypeStoredWarehouse:
		return PropertyTypeStoredWarehouse
	default:
		return strings.TrimSpace(v)
	}
}

// NormalizePropertyTypeCatalogFilter нормализует query propertyType для фильтра каталога.
func NormalizePropertyTypeCatalogFilter(v string) string {
	return NormalizePropertySubcategoryAPI(v)
}

// IsPropertyTypeAllowedForCategory проверяет связку category + property_type (уже в формате БД).
func IsPropertyTypeAllowedForCategory(category, propertyType string) bool {
	category = NormalizeCategoryStored(category)
	propertyType = NormalizePropertySubcategoryAPI(propertyType)
	switch category {
	case CategoryStoredResidential:
		switch propertyType {
		case PropertyTypeStoredApartment, PropertyTypeStoredRoom, PropertyTypeStoredHouse,
			PropertyTypeStoredStudio, PropertyTypeStoredCottage:
			return true
		}
	case CategoryStoredCommercial:
		switch propertyType {
		case PropertyTypeStoredOffice, PropertyTypeStoredCoworking, PropertyTypeStoredBuilding,
			PropertyTypeStoredWarehouse:
			return true
		}
	}
	return false
}

// PropertyTypeRequiresRoomCount — для студии и коммерческой категории rooms не обязателен.
func PropertyTypeRequiresRoomCount(category, propertyType string) bool {
	if NormalizeCategoryStored(category) == CategoryStoredCommercial {
		return false
	}
	return NormalizePropertySubcategoryAPI(propertyType) != PropertyTypeStoredStudio
}

// IsValidPropertySubcategoryAPI — допустимые значения subcategory в multipart/JSON создания.
func IsValidPropertySubcategoryAPI(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case PropertySubcategoryApartment, PropertySubcategoryRoom, PropertySubcategoryHouse,
		PropertySubcategoryStudio, PropertySubcategoryCottage,
		PropertySubcategoryOffice, PropertySubcategoryCoworking, PropertySubcategoryBuilding,
		PropertySubcategoryWarehouse:
		return true
	default:
		return false
	}
}

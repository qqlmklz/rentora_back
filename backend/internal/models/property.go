package models

// Property — снимок карточки объявления (каталог, избранное, вложение в GET /api/profile/requests как поле "property").
type Property struct {
	ID           int      `json:"id"`
	Title        string   `json:"title"`
	Price        int      `json:"price"`
	PropertyType string   `json:"propertyType"`
	Rooms        int      `json:"rooms"`
	TotalArea    float64  `json:"totalArea"`
	City         string   `json:"city"`
	District     string   `json:"district"`
	Photos       []string `json:"photos"`
	IsArchived   bool     `json:"isArchived"`
}

type ProfilePropertiesResponse struct {
	ActiveListings   []Property `json:"activeListings"`
	ArchivedListings []Property `json:"archivedListings"`
}

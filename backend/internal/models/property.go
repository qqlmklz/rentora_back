package models

// Property represents a listing used in catalog and favorites.
// This struct is focused on card data; extend when details page is added.
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
}

package models

// Property represents a listing used in catalog and favorites.
// This struct is focused on card data; extend when details page is added.
type Property struct {
	ID           int      `json:"id"`
	Title        string   `json:"title"`
	Category     string   `json:"category,omitempty"`
	Price        int      `json:"price"`
	PropertyType string   `json:"property_type"`
	Rooms        int      `json:"rooms"`
	Area         float64  `json:"area"`
	City         string   `json:"city"`
	District     string   `json:"district"`
	Image        *string  `json:"image,omitempty"`
}

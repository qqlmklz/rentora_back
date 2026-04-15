package models

// Базовая модель объявления для каталога и избранного.
// Здесь только поля карточки; если нужно, для страницы деталей можно расширить.
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

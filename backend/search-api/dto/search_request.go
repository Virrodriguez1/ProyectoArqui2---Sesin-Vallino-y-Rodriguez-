package dto

// SearchRequest representa los parámetros de búsqueda de propiedades
type SearchRequest struct {
	Query     string  `json:"query" form:"query"`
	City      string  `json:"city" form:"city"`
	Country   string  `json:"country" form:"country"`
	MinPrice  float64 `json:"min_price" form:"min_price"`
	MaxPrice  float64 `json:"max_price" form:"max_price"`
	Bedrooms  int     `json:"bedrooms" form:"bedrooms"`
	Bathrooms int     `json:"bathrooms" form:"bathrooms"`
	MinGuests int     `json:"min_guests" form:"min_guests"`
	Page      int     `json:"page" form:"page"`
	PageSize  int     `json:"page_size" form:"page_size"`
	SortBy    string  `json:"sort_by" form:"sort_by"`
	SortOrder string  `json:"sort_order" form:"sort_order"`
}


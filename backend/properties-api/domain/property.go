package domain

import "time"

// Property representa una propiedad de alquiler tipo Airbnb
type Property struct {
	ID            string    `json:"id"`
	Title         string    `json:"title"`
	Description   string    `json:"description"`
	City          string    `json:"city"`
	Country       string    `json:"country"`
	PricePerNight float64   `json:"price_per_night"`
	Bedrooms      int       `json:"bedrooms"`
	Bathrooms     int       `json:"bathrooms"`
	MaxGuests     int       `json:"max_guests"`
	Images        []string  `json:"images"`
	OwnerID       uint      `json:"owner_id"`
	Available     bool      `json:"available"`
	CreatedAt     time.Time `json:"created_at"`
}


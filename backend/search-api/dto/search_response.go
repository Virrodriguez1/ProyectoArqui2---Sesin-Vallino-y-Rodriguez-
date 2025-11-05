package dto

import "backend/properties-api/domain"

// SearchResponse representa la respuesta de una búsqueda de propiedades
type SearchResponse struct {
	Results     []domain.Property `json:"results"`
	TotalResults int             `json:"total_results"`
	Page        int              `json:"page"`
	PageSize    int              `json:"page_size"`
	TotalPages  int              `json:"total_pages"`
}

// ErrorResponse representa una respuesta de error
type ErrorResponse struct {
	Error string `json:"error"`
	Code  int    `json:"code"`
}


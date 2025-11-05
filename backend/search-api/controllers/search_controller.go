package controllers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"backend/search-api/dto"
	"backend/search-api/services"
)

// SearchController maneja las peticiones HTTP de búsqueda
type SearchController struct {
	service services.SearchService
}

// NewSearchController crea una nueva instancia de SearchController
func NewSearchController(service services.SearchService) *SearchController {
	return &SearchController{
		service: service,
	}
}

// Search maneja las peticiones de búsqueda de propiedades
func (c *SearchController) Search(w http.ResponseWriter, r *http.Request) {
	// Solo permitir método GET
	if r.Method != http.MethodGet {
		writeErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parsear query parameters a SearchRequest
	request, err := parseSearchRequest(r)
	if err != nil {
		log.Printf("Error parsing search request: %v", err)
		writeErrorResponse(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Aplicar valores por defecto
	applyDefaults(request)

	// Validar parámetros
	if err := validateSearchRequest(request); err != nil {
		log.Printf("Validation error: %v", err)
		writeErrorResponse(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Llamar al servicio
	ctx := r.Context()
	response, err := c.service.Search(ctx, *request)
	if err != nil {
		log.Printf("Error searching properties: %v", err)
		writeErrorResponse(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Escribir respuesta exitosa
	writeJSONResponse(w, response, http.StatusOK)
}

// parseSearchRequest parsea los query parameters a SearchRequest
func parseSearchRequest(r *http.Request) (*dto.SearchRequest, error) {
	query := r.URL.Query()
	
	request := &dto.SearchRequest{
		Query:     query.Get("query"),
		City:      query.Get("city"),
		Country:   query.Get("country"),
		SortBy:    query.Get("sort_by"),
		SortOrder: query.Get("sort_order"),
	}

	// Parsear MinPrice
	if minPriceStr := query.Get("min_price"); minPriceStr != "" {
		minPrice, err := strconv.ParseFloat(minPriceStr, 64)
		if err != nil {
			return nil, err
		}
		request.MinPrice = minPrice
	}

	// Parsear MaxPrice
	if maxPriceStr := query.Get("max_price"); maxPriceStr != "" {
		maxPrice, err := strconv.ParseFloat(maxPriceStr, 64)
		if err != nil {
			return nil, err
		}
		request.MaxPrice = maxPrice
	}

	// Parsear Bedrooms
	if bedroomsStr := query.Get("bedrooms"); bedroomsStr != "" {
		bedrooms, err := strconv.Atoi(bedroomsStr)
		if err != nil {
			return nil, err
		}
		request.Bedrooms = bedrooms
	}

	// Parsear Bathrooms
	if bathroomsStr := query.Get("bathrooms"); bathroomsStr != "" {
		bathrooms, err := strconv.Atoi(bathroomsStr)
		if err != nil {
			return nil, err
		}
		request.Bathrooms = bathrooms
	}

	// Parsear MinGuests
	if minGuestsStr := query.Get("min_guests"); minGuestsStr != "" {
		minGuests, err := strconv.Atoi(minGuestsStr)
		if err != nil {
			return nil, err
		}
		request.MinGuests = minGuests
	}

	// Parsear Page
	if pageStr := query.Get("page"); pageStr != "" {
		page, err := strconv.Atoi(pageStr)
		if err != nil {
			return nil, err
		}
		request.Page = page
	}

	// Parsear PageSize
	if pageSizeStr := query.Get("page_size"); pageSizeStr != "" {
		pageSize, err := strconv.Atoi(pageSizeStr)
		if err != nil {
			return nil, err
		}
		request.PageSize = pageSize
	}

	return request, nil
}

// applyDefaults aplica valores por defecto a los parámetros de búsqueda
func applyDefaults(request *dto.SearchRequest) {
	if request.Page < 1 {
		request.Page = 1
	}
	if request.PageSize < 1 {
		request.PageSize = 10
	}
	if request.SortBy == "" {
		request.SortBy = "price_per_night"
	}
	if request.SortOrder == "" {
		request.SortOrder = "asc"
	}
}

// validateSearchRequest valida los parámetros de búsqueda
func validateSearchRequest(request *dto.SearchRequest) error {
	// Validar Page >= 1
	if request.Page < 1 {
		return &ValidationError{Message: "Page must be >= 1"}
	}

	// Validar PageSize > 0 y <= 100
	if request.PageSize < 1 {
		return &ValidationError{Message: "PageSize must be > 0"}
	}
	if request.PageSize > 100 {
		return &ValidationError{Message: "PageSize must be <= 100"}
	}

	// Validar SortOrder si está presente
	if request.SortOrder != "" && request.SortOrder != "asc" && request.SortOrder != "desc" {
		return &ValidationError{Message: "SortOrder must be 'asc' or 'desc'"}
	}

	// Validar rango de precio
	if request.MinPrice < 0 {
		return &ValidationError{Message: "MinPrice cannot be negative"}
	}
	if request.MaxPrice < 0 {
		return &ValidationError{Message: "MaxPrice cannot be negative"}
	}
	if request.MinPrice > 0 && request.MaxPrice > 0 && request.MinPrice > request.MaxPrice {
		return &ValidationError{Message: "MinPrice cannot be greater than MaxPrice"}
	}

	// Validar bedrooms, bathrooms, min_guests
	if request.Bedrooms < 0 {
		return &ValidationError{Message: "Bedrooms cannot be negative"}
	}
	if request.Bathrooms < 0 {
		return &ValidationError{Message: "Bathrooms cannot be negative"}
	}
	if request.MinGuests < 0 {
		return &ValidationError{Message: "MinGuests cannot be negative"}
	}

	return nil
}

// writeJSONResponse escribe una respuesta JSON exitosa
func writeJSONResponse(w http.ResponseWriter, data interface{}, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Error encoding JSON response: %v", err)
		// Si ya escribimos el status code, no podemos cambiarlo
		// Intentar escribir un error simple
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
	}
}

// writeErrorResponse escribe una respuesta de error JSON
func writeErrorResponse(w http.ResponseWriter, message string, statusCode int) {
	errorResponse := dto.ErrorResponse{
		Error: message,
		Code:  statusCode,
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	
	if err := json.NewEncoder(w).Encode(errorResponse); err != nil {
		log.Printf("Error encoding error response: %v", err)
		// Si falla, escribir un error simple
		http.Error(w, message, statusCode)
	}
}

// ValidationError representa un error de validación
type ValidationError struct {
	Message string
}

// Error implementa la interfaz error
func (e *ValidationError) Error() string {
	return e.Message
}


package services

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"backend/properties-api/domain"
	"backend/search-api/dto"
	"backend/search-api/repositories"
)

// SearchService define la interfaz para operaciones de búsqueda
type SearchService interface {
	Search(ctx context.Context, request dto.SearchRequest) (*dto.SearchResponse, error)
	IndexProperty(ctx context.Context, property domain.Property) error
	UpdateProperty(ctx context.Context, property domain.Property) error
	DeleteProperty(ctx context.Context, propertyID string) error
	FetchPropertyFromAPI(propertyID string) (*domain.Property, error)
}

// searchService implementa SearchService
type searchService struct {
	solrRepo        repositories.SolrRepository
	cacheRepo       repositories.CacheRepository
	propertiesAPIURL string
	httpClient      *http.Client
}

// NewSearchService crea una nueva instancia de SearchService
func NewSearchService(solrRepo repositories.SolrRepository, cacheRepo repositories.CacheRepository, apiURL string) SearchService {
	return &searchService{
		solrRepo:         solrRepo,
		cacheRepo:        cacheRepo,
		propertiesAPIURL: strings.TrimSuffix(apiURL, "/"),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// generateCacheKey genera una clave de caché basada en los parámetros del request
func (s *searchService) generateCacheKey(request dto.SearchRequest) string {
	// Crear una representación del request para hashear
	keyParts := []string{
		fmt.Sprintf("query:%s", request.Query),
		fmt.Sprintf("city:%s", request.City),
		fmt.Sprintf("country:%s", request.Country),
		fmt.Sprintf("min_price:%.2f", request.MinPrice),
		fmt.Sprintf("max_price:%.2f", request.MaxPrice),
		fmt.Sprintf("bedrooms:%d", request.Bedrooms),
		fmt.Sprintf("bathrooms:%d", request.Bathrooms),
		fmt.Sprintf("min_guests:%d", request.MinGuests),
		fmt.Sprintf("page:%d", request.Page),
		fmt.Sprintf("page_size:%d", request.PageSize),
		fmt.Sprintf("sort_by:%s", request.SortBy),
		fmt.Sprintf("sort_order:%s", request.SortOrder),
	}
	
	keyString := strings.Join(keyParts, "|")
	hash := md5.Sum([]byte(keyString))
	return fmt.Sprintf("search:%x", hash)
}

// Search implementa la búsqueda con caché
func (s *searchService) Search(ctx context.Context, request dto.SearchRequest) (*dto.SearchResponse, error) {
	// Validar y aplicar valores por defecto
	if err := s.validateSearchRequest(&request); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Generar clave de caché (después de aplicar valores por defecto)
	cacheKey := s.generateCacheKey(request)

	// 1. Consultar caché primero
	log.Printf("Search: Checking cache for key=%s", cacheKey)
	properties, total, found := s.cacheRepo.Get(cacheKey)
	if found {
		log.Printf("Search: Cache HIT for key=%s", cacheKey)
		// Calcular TotalPages (pageSize ya tiene valor por defecto aplicado)
		pageSize := request.PageSize
		totalPages := (total + pageSize - 1) / pageSize // Redondeo hacia arriba
		
		return &dto.SearchResponse{
			Results:     properties,
			TotalResults: total,
			Page:        request.Page,
			PageSize:    pageSize,
			TotalPages:  totalPages,
		}, nil
	}

	log.Printf("Search: Cache MISS for key=%s, querying Solr", cacheKey)

	// 2. Si no hay hit, consultar Solr
	properties, total, err := s.solrRepo.Search(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("error searching in Solr: %w", err)
	}

	log.Printf("Search: Solr returned %d results, total=%d", len(properties), total)

	// 3. Guardar resultado en caché
	s.cacheRepo.Set(cacheKey, properties, total, 10*time.Minute)
	log.Printf("Search: Results cached with key=%s", cacheKey)

	// 4. Calcular TotalPages (pageSize ya tiene valor por defecto aplicado)
	pageSize := request.PageSize
	totalPages := (total + pageSize - 1) / pageSize // Redondeo hacia arriba

	// 5. Retornar SearchResponse completo
	return &dto.SearchResponse{
		Results:     properties,
		TotalResults: total,
		Page:        request.Page,
		PageSize:    pageSize,
		TotalPages:  totalPages,
	}, nil
}

// IndexProperty indexa una propiedad en Solr e invalida caché
func (s *searchService) IndexProperty(ctx context.Context, property domain.Property) error {
	// Validar property
	if err := s.validateProperty(&property); err != nil {
		return fmt.Errorf("invalid property: %w", err)
	}

	log.Printf("IndexProperty: Indexing property ID=%s", property.ID)

	// Indexar en Solr
	if err := s.solrRepo.IndexProperty(ctx, property); err != nil {
		return fmt.Errorf("error indexing property in Solr: %w", err)
	}

	log.Printf("IndexProperty: Property ID=%s indexed successfully", property.ID)

	// Invalidar caché (eliminar todas las claves de búsqueda)
	// Nota: En una implementación real, podrías querer invalidar solo claves relacionadas
	// Por simplicidad, invalidamos todas las búsquedas
	s.invalidateCache()

	return nil
}

// UpdateProperty actualiza una propiedad en Solr e invalida caché
func (s *searchService) UpdateProperty(ctx context.Context, property domain.Property) error {
	// Validar property
	if err := s.validateProperty(&property); err != nil {
		return fmt.Errorf("invalid property: %w", err)
	}

	log.Printf("UpdateProperty: Updating property ID=%s", property.ID)

	// Actualizar en Solr
	if err := s.solrRepo.UpdateProperty(ctx, property); err != nil {
		return fmt.Errorf("error updating property in Solr: %w", err)
	}

	log.Printf("UpdateProperty: Property ID=%s updated successfully", property.ID)

	// Invalidar caché
	s.invalidateCache()

	return nil
}

// DeleteProperty elimina una propiedad de Solr e invalida caché
func (s *searchService) DeleteProperty(ctx context.Context, propertyID string) error {
	// Validar propertyID
	if propertyID == "" {
		return fmt.Errorf("property ID cannot be empty")
	}

	log.Printf("DeleteProperty: Deleting property ID=%s", propertyID)

	// Eliminar de Solr
	if err := s.solrRepo.DeleteProperty(ctx, propertyID); err != nil {
		return fmt.Errorf("error deleting property from Solr: %w", err)
	}

	log.Printf("DeleteProperty: Property ID=%s deleted successfully", propertyID)

	// Invalidar caché
	s.invalidateCache()

	return nil
}

// FetchPropertyFromAPI obtiene una propiedad desde la API de propiedades
func (s *searchService) FetchPropertyFromAPI(propertyID string) (*domain.Property, error) {
	// Validar propertyID
	if propertyID == "" {
		return nil, fmt.Errorf("property ID cannot be empty")
	}

	// Construir URL
	url := fmt.Sprintf("%s/properties/%s", s.propertiesAPIURL, propertyID)

	log.Printf("FetchPropertyFromAPI: Fetching property ID=%s from %s", propertyID, url)

	// Crear request HTTP
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Ejecutar request
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error executing request: %w", err)
	}
	defer resp.Body.Close()

	// Verificar status code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("properties API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Leer respuesta
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %w", err)
	}

	// Parsear respuesta JSON
	var property domain.Property
	if err := json.Unmarshal(body, &property); err != nil {
		return nil, fmt.Errorf("error parsing response: %w", err)
	}

	log.Printf("FetchPropertyFromAPI: Property ID=%s fetched successfully", propertyID)

	return &property, nil
}

// validateSearchRequest valida los parámetros de búsqueda
func (s *searchService) validateSearchRequest(request *dto.SearchRequest) error {
	// Aplicar valores por defecto
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

	// Validar sort order
	if request.SortOrder != "asc" && request.SortOrder != "desc" {
		return fmt.Errorf("invalid sort_order: must be 'asc' or 'desc'")
	}

	// Validar rango de precio
	if request.MinPrice < 0 {
		return fmt.Errorf("min_price cannot be negative")
	}
	if request.MaxPrice < 0 {
		return fmt.Errorf("max_price cannot be negative")
	}
	if request.MinPrice > 0 && request.MaxPrice > 0 && request.MinPrice > request.MaxPrice {
		return fmt.Errorf("min_price cannot be greater than max_price")
	}

	// Validar bedrooms, bathrooms, min_guests
	if request.Bedrooms < 0 {
		return fmt.Errorf("bedrooms cannot be negative")
	}
	if request.Bathrooms < 0 {
		return fmt.Errorf("bathrooms cannot be negative")
	}
	if request.MinGuests < 0 {
		return fmt.Errorf("min_guests cannot be negative")
	}

	return nil
}

// validateProperty valida una propiedad
func (s *searchService) validateProperty(property *domain.Property) error {
	if property.ID == "" {
		return fmt.Errorf("property ID cannot be empty")
	}
	if property.Title == "" {
		return fmt.Errorf("property title cannot be empty")
	}
	if property.City == "" {
		return fmt.Errorf("property city cannot be empty")
	}
	if property.Country == "" {
		return fmt.Errorf("property country cannot be empty")
	}
	if property.PricePerNight < 0 {
		return fmt.Errorf("property price_per_night cannot be negative")
	}
	if property.Bedrooms < 0 {
		return fmt.Errorf("property bedrooms cannot be negative")
	}
	if property.Bathrooms < 0 {
		return fmt.Errorf("property bathrooms cannot be negative")
	}
	if property.MaxGuests < 0 {
		return fmt.Errorf("property max_guests cannot be negative")
	}
	return nil
}

// invalidateCache invalida todas las claves de caché relacionadas con búsquedas
// Nota: En una implementación real, podrías mantener un registro de claves o usar un patrón de invalidadión más sofisticado
func (s *searchService) invalidateCache() {
	// Por simplicidad, no podemos eliminar todas las claves sin conocerlas
	// En una implementación real, podrías:
	// 1. Mantener un registro de claves activas
	// 2. Usar un prefijo y eliminar todas las claves con ese prefijo (si Memcached lo soporta)
	// 3. Usar un timestamp de versión en las claves de caché
	log.Printf("Cache invalidation: Note that all search cache keys should be invalidated")
	// Por ahora, la invalidación se hará naturalmente cuando expire el TTL
}


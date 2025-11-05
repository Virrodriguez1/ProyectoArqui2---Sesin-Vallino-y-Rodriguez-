package repositories

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"backend/properties-api/domain"
	"backend/search-api/dto"
)

// SolrRepository define la interfaz para operaciones con Solr
type SolrRepository interface {
	Search(ctx context.Context, request dto.SearchRequest) ([]domain.Property, int, error)
	IndexProperty(ctx context.Context, property domain.Property) error
	UpdateProperty(ctx context.Context, property domain.Property) error
	DeleteProperty(ctx context.Context, propertyID string) error
}

// solrRepository implementa SolrRepository
type solrRepository struct {
	solrURL    string
	httpClient *http.Client
}

// NewSolrRepository crea una nueva instancia de SolrRepository
func NewSolrRepository(solrURL string) SolrRepository {
	return &solrRepository{
		solrURL: solrURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SolrSearchResponse representa la respuesta de búsqueda de Solr
type solrSearchResponse struct {
	Response struct {
		NumFound int                      `json:"numFound"`
		Docs     []map[string]interface{} `json:"docs"`
	} `json:"response"`
}

// SolrUpdateResponse representa la respuesta de actualización de Solr
type solrUpdateResponse struct {
	ResponseHeader struct {
		Status int `json:"status"`
	} `json:"responseHeader"`
}

// Search implementa la búsqueda en Solr
func (r *solrRepository) Search(ctx context.Context, request dto.SearchRequest) ([]domain.Property, int, error) {
	// Construir URL base
	baseURL := strings.TrimSuffix(r.solrURL, "/")
	searchURL := fmt.Sprintf("%s/select", baseURL)

	// Construir parámetros de query
	params := url.Values{}
	
	// Construir query de texto
	var queryParts []string
	if request.Query != "" {
		queryParts = append(queryParts, fmt.Sprintf("(title:*%s* OR city:*%s* OR country:*%s*)", 
			escapeSolrQuery(request.Query), 
			escapeSolrQuery(request.Query), 
			escapeSolrQuery(request.Query)))
	}
	if len(queryParts) == 0 {
		params.Set("q", "*:*")
	} else {
		params.Set("q", strings.Join(queryParts, " AND "))
	}

	// Construir filtros (fq)
	var filters []string

	// Filtro por rango de precio
	if request.MinPrice > 0 || request.MaxPrice > 0 {
		minPrice := request.MinPrice
		if minPrice == 0 {
			minPrice = 0
		}
		maxPrice := request.MaxPrice
		if maxPrice == 0 {
			maxPrice = 999999
		}
		filters = append(filters, fmt.Sprintf("price_per_night:[%f TO %f]", minPrice, maxPrice))
	}

	// Filtro por bedrooms
	if request.Bedrooms > 0 {
		filters = append(filters, fmt.Sprintf("bedrooms:%d", request.Bedrooms))
	}

	// Filtro por bathrooms
	if request.Bathrooms > 0 {
		filters = append(filters, fmt.Sprintf("bathrooms:%d", request.Bathrooms))
	}

	// Filtro por min_guests
	if request.MinGuests > 0 {
		filters = append(filters, fmt.Sprintf("max_guests:[%d TO *]", request.MinGuests))
	}

	// Filtro por city
	if request.City != "" {
		filters = append(filters, fmt.Sprintf("city:\"%s\"", escapeSolrQuery(request.City)))
	}

	// Filtro por country
	if request.Country != "" {
		filters = append(filters, fmt.Sprintf("country:\"%s\"", escapeSolrQuery(request.Country)))
	}

	// Agregar filtros a fq
	if len(filters) > 0 {
		for _, filter := range filters {
			params.Add("fq", filter)
		}
	}

	// Paginación
	page := request.Page
	if page < 1 {
		page = 1
	}
	pageSize := request.PageSize
	if pageSize < 1 {
		pageSize = 10
	}
	start := (page - 1) * pageSize
	params.Set("start", strconv.Itoa(start))
	params.Set("rows", strconv.Itoa(pageSize))

	// Sorting
	sortBy := request.SortBy
	if sortBy == "" {
		sortBy = "price_per_night"
	}
	sortOrder := request.SortOrder
	if sortOrder == "" {
		sortOrder = "asc"
	}
	params.Set("sort", fmt.Sprintf("%s %s", sortBy, sortOrder))

	// Formato de respuesta
	params.Set("wt", "json")

	// Construir URL completa
	fullURL := fmt.Sprintf("%s?%s", searchURL, params.Encode())

	// Crear request HTTP
	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("error creating request: %w", err)
	}

	// Ejecutar request
	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("error executing request: %w", err)
	}
	defer resp.Body.Close()

	// Verificar status code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, 0, fmt.Errorf("solr returned status %d: %s", resp.StatusCode, string(body))
	}

	// Leer respuesta
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, fmt.Errorf("error reading response: %w", err)
	}

	// Parsear respuesta JSON
	var solrResp solrSearchResponse
	if err := json.Unmarshal(body, &solrResp); err != nil {
		return nil, 0, fmt.Errorf("error parsing response: %w", err)
	}

	// Convertir docs a domain.Property
	properties := make([]domain.Property, 0, len(solrResp.Response.Docs))
	for _, doc := range solrResp.Response.Docs {
		property := r.mapDocToProperty(doc)
		properties = append(properties, property)
	}

	return properties, solrResp.Response.NumFound, nil
}

// IndexProperty indexa una propiedad en Solr
func (r *solrRepository) IndexProperty(ctx context.Context, property domain.Property) error {
	baseURL := strings.TrimSuffix(r.solrURL, "/")
	updateURL := fmt.Sprintf("%s/update/json/docs", baseURL)

	// Convertir property a JSON
	propertyJSON, err := json.Marshal(property)
	if err != nil {
		return fmt.Errorf("error marshaling property: %w", err)
	}

	// Crear request HTTP
	req, err := http.NewRequestWithContext(ctx, "POST", updateURL, bytes.NewBuffer(propertyJSON))
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Ejecutar request
	resp, err := r.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("error executing request: %w", err)
	}
	defer resp.Body.Close()

	// Verificar status code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("solr returned status %d: %s", resp.StatusCode, string(body))
	}

	// Leer respuesta
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response: %w", err)
	}

	// Parsear respuesta
	var updateResp solrUpdateResponse
	if err := json.Unmarshal(body, &updateResp); err != nil {
		return fmt.Errorf("error parsing response: %w", err)
	}

	if updateResp.ResponseHeader.Status != 0 {
		return fmt.Errorf("solr update failed with status %d", updateResp.ResponseHeader.Status)
	}

	// Hacer commit
	return r.commit(ctx)
}

// UpdateProperty actualiza una propiedad en Solr
func (r *solrRepository) UpdateProperty(ctx context.Context, property domain.Property) error {
	// Update es similar a Index en Solr
	return r.IndexProperty(ctx, property)
}

// DeleteProperty elimina una propiedad de Solr
func (r *solrRepository) DeleteProperty(ctx context.Context, propertyID string) error {
	baseURL := strings.TrimSuffix(r.solrURL, "/")
	updateURL := fmt.Sprintf("%s/update", baseURL)

	// Construir comando de delete
	deleteCmd := map[string]interface{}{
		"delete": map[string]string{
			"id": propertyID,
		},
	}

	deleteJSON, err := json.Marshal(deleteCmd)
	if err != nil {
		return fmt.Errorf("error marshaling delete command: %w", err)
	}

	// Crear request HTTP
	req, err := http.NewRequestWithContext(ctx, "POST", updateURL, bytes.NewBuffer(deleteJSON))
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Ejecutar request
	resp, err := r.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("error executing request: %w", err)
	}
	defer resp.Body.Close()

	// Verificar status code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("solr returned status %d: %s", resp.StatusCode, string(body))
	}

	// Leer respuesta
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response: %w", err)
	}

	// Parsear respuesta
	var updateResp solrUpdateResponse
	if err := json.Unmarshal(body, &updateResp); err != nil {
		return fmt.Errorf("error parsing response: %w", err)
	}

	if updateResp.ResponseHeader.Status != 0 {
		return fmt.Errorf("solr delete failed with status %d", updateResp.ResponseHeader.Status)
	}

	// Hacer commit
	return r.commit(ctx)
}

// commit realiza un commit en Solr
func (r *solrRepository) commit(ctx context.Context) error {
	baseURL := strings.TrimSuffix(r.solrURL, "/")
	updateURL := fmt.Sprintf("%s/update", baseURL)

	commitCmd := map[string]interface{}{
		"commit": map[string]interface{}{},
	}

	commitJSON, err := json.Marshal(commitCmd)
	if err != nil {
		return fmt.Errorf("error marshaling commit command: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", updateURL, bytes.NewBuffer(commitJSON))
	if err != nil {
		return fmt.Errorf("error creating commit request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("error executing commit: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("solr commit returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// mapDocToProperty convierte un documento de Solr a domain.Property
func (r *solrRepository) mapDocToProperty(doc map[string]interface{}) domain.Property {
	property := domain.Property{}

	if id, ok := doc["id"].(string); ok {
		property.ID = id
	}
	if title, ok := doc["title"].(string); ok {
		property.Title = title
	}
	if desc, ok := doc["description"].(string); ok {
		property.Description = desc
	}
	if city, ok := doc["city"].(string); ok {
		property.City = city
	}
	if country, ok := doc["country"].(string); ok {
		property.Country = country
	}
	if price, ok := doc["price_per_night"].(float64); ok {
		property.PricePerNight = price
	}
	if bedrooms, ok := doc["bedrooms"].(float64); ok {
		property.Bedrooms = int(bedrooms)
	}
	if bathrooms, ok := doc["bathrooms"].(float64); ok {
		property.Bathrooms = int(bathrooms)
	}
	if maxGuests, ok := doc["max_guests"].(float64); ok {
		property.MaxGuests = int(maxGuests)
	}
	if images, ok := doc["images"].([]interface{}); ok {
		property.Images = make([]string, 0, len(images))
		for _, img := range images {
			if imgStr, ok := img.(string); ok {
				property.Images = append(property.Images, imgStr)
			}
		}
	}
	if ownerID, ok := doc["owner_id"].(float64); ok {
		property.OwnerID = uint(ownerID)
	}
	if available, ok := doc["available"].(bool); ok {
		property.Available = available
	}
	if createdAt, ok := doc["created_at"].(string); ok {
		if t, err := time.Parse(time.RFC3339, createdAt); err == nil {
			property.CreatedAt = t
		}
	}

	return property
}

// escapeSolrQuery escapa caracteres especiales en la query de Solr
func escapeSolrQuery(query string) string {
	// Escapar caracteres especiales de Solr
	query = strings.ReplaceAll(query, "\\", "\\\\")
	query = strings.ReplaceAll(query, "+", "\\+")
	query = strings.ReplaceAll(query, "-", "\\-")
	query = strings.ReplaceAll(query, "&", "\\&")
	query = strings.ReplaceAll(query, "|", "\\|")
	query = strings.ReplaceAll(query, "!", "\\!")
	query = strings.ReplaceAll(query, "(", "\\(")
	query = strings.ReplaceAll(query, ")", "\\)")
	query = strings.ReplaceAll(query, "{", "\\{")
	query = strings.ReplaceAll(query, "}", "\\}")
	query = strings.ReplaceAll(query, "[", "\\[")
	query = strings.ReplaceAll(query, "]", "\\]")
	query = strings.ReplaceAll(query, "^", "\\^")
	query = strings.ReplaceAll(query, "\"", "\\\"")
	query = strings.ReplaceAll(query, "~", "\\~")
	query = strings.ReplaceAll(query, "*", "\\*")
	query = strings.ReplaceAll(query, "?", "\\?")
	query = strings.ReplaceAll(query, ":", "\\:")
	return query
}


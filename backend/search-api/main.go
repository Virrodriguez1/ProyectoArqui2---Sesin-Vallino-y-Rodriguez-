package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"backend/search-api/config"
	"backend/search-api/consumers"
	"backend/search-api/controllers"
	"backend/search-api/repositories"
	"backend/search-api/services"
)

func main() {
	log.Println("Starting Search API...")

	// a. Cargar configuración
	cfg := config.LoadConfig()
	log.Printf("Configuration loaded: Port=%s, SolrURL=%s, MemcachedHost=%s", 
		cfg.Port, cfg.SolrURL, cfg.MemcachedHost)

	// b. Inicializar repositorios
	log.Println("Initializing repositories...")
	solrRepo := repositories.NewSolrRepository(cfg.SolrURL)
	log.Println("Solr repository initialized")

	cacheRepo := repositories.NewCacheRepository(cfg.MemcachedHost)
	log.Println("Cache repository initialized")

	// c. Inicializar servicio
	log.Println("Initializing service...")
	searchService := services.NewSearchService(solrRepo, cacheRepo, cfg.PropertiesAPIURL)
	log.Println("Search service initialized")

	// d. Inicializar controlador
	log.Println("Initializing controller...")
	searchController := controllers.NewSearchController(searchService)
	log.Println("Search controller initialized")

	// e. Inicializar y arrancar consumidor de RabbitMQ en una goroutine
	log.Println("Initializing RabbitMQ consumer...")
	consumer, err := consumers.NewRabbitMQConsumer(cfg.RabbitMQURL, "properties_queue", searchService)
	if err != nil {
		log.Fatalf("Failed to create RabbitMQ consumer: %v", err)
	}
	log.Println("RabbitMQ consumer created")

	// Arrancar consumidor en goroutine
	go func() {
		if err := consumer.Start(); err != nil {
			log.Printf("Error starting RabbitMQ consumer: %v", err)
		}
	}()
	log.Println("RabbitMQ consumer started")

	// f. Configurar router HTTP
	log.Println("Configuring HTTP routes...")
	
	// Health check endpoint
	http.HandleFunc("/health", healthHandler)
	log.Println("Route registered: GET /health")

	// Search endpoint con middleware CORS
	http.HandleFunc("/search", corsMiddleware(searchController.Search))
	log.Println("Route registered: GET /search")

	// g. Crear servidor HTTP
	server := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: nil, // Usa DefaultServeMux
	}

	// h. Iniciar servidor HTTP en goroutine
	go func() {
		log.Printf("Starting HTTP server on port %s...", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	log.Printf("Search API started successfully on port %s", cfg.Port)

	// i. Manejar graceful shutdown con signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down Search API...")

	// Crear contexto con timeout para shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Cerrar servidor HTTP
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Error shutting down server: %v", err)
	} else {
		log.Println("HTTP server shut down successfully")
	}

	// Cerrar consumidor RabbitMQ
	if err := consumer.Close(); err != nil {
		log.Printf("Error closing RabbitMQ consumer: %v", err)
	} else {
		log.Println("RabbitMQ consumer closed successfully")
	}

	log.Println("Search API shut down complete")
}

// healthHandler maneja las peticiones de health check
func healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	
	response := map[string]string{
		"status": "ok",
	}
	
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding health response: %v", err)
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
	}
}

// corsMiddleware agrega headers CORS a las respuestas
func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Max-Age", "3600")

		// Handle preflight requests
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		// Call the next handler
		next(w, r)
	}
}


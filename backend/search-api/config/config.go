package config

import "os"

// Config contiene la configuración de la aplicación
type Config struct {
	SolrURL         string
	MemcachedHost   string
	RabbitMQURL     string
	PropertiesAPIURL string
	Port            string
}

// LoadConfig carga la configuración desde variables de entorno con valores por defecto
func LoadConfig() *Config {
	cfg := &Config{
		SolrURL:         getEnv("SOLR_URL", "http://localhost:8983/solr/properties"),
		MemcachedHost:   getEnv("MEMCACHED_HOST", "localhost:11211"),
		RabbitMQURL:     getEnv("RABBITMQ_URL", "amqp://admin:admin@localhost:5672/"),
		PropertiesAPIURL: getEnv("PROPERTIES_API_URL", "http://localhost:8081"),
		Port:            getEnv("PORT", "8082"),
	}
	return cfg
}

// getEnv obtiene una variable de entorno o retorna un valor por defecto
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}


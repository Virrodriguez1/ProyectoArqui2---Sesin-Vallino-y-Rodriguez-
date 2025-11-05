package repositories

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/karlseguin/ccache/v3"
	"backend/properties-api/domain"
)

// CacheRepository define la interfaz para operaciones de caché
type CacheRepository interface {
	Get(key string) ([]domain.Property, int, bool)
	Set(key string, properties []domain.Property, total int, ttl time.Duration)
	Delete(key string)
}

// cacheData representa los datos almacenados en caché
type cacheData struct {
	Properties []domain.Property `json:"properties"`
	Total      int               `json:"total"`
}

// cacheRepository implementa CacheRepository con dos niveles
type cacheRepository struct {
	localCache     *ccache.Cache[string, *cacheData]
	memcachedClient *memcache.Client
}

// NewCacheRepository crea una nueva instancia de CacheRepository
func NewCacheRepository(memcachedHost string) CacheRepository {
	// Inicializar ccache local con configuración por defecto
	localCache := ccache.New(ccache.Configure[string, *cacheData]().MaxSize(1000))

	// Conectar con Memcached
	memcachedClient := memcache.New(memcachedHost)
	
	log.Printf("Cache repository initialized with Memcached at %s", memcachedHost)

	return &cacheRepository{
		localCache:     localCache,
		memcachedClient: memcachedClient,
	}
}

// Get obtiene datos del caché (primero local, luego Memcached)
func (r *cacheRepository) Get(key string) ([]domain.Property, int, bool) {
	// 1. Buscar en caché local primero
	item := r.localCache.Get(key)
	if item != nil && !item.Expired() {
		data := item.Value()
		log.Printf("Cache HIT (local): key=%s", key)
		return data.Properties, data.Total, true
	}

	// 2. Si no está en local, buscar en Memcached
	memcachedItem, err := r.memcachedClient.Get(key)
	if err != nil {
		if err == memcache.ErrCacheMiss {
			log.Printf("Cache MISS: key=%s", key)
			return nil, 0, false
		}
		log.Printf("Error getting from Memcached: key=%s, error=%v", key, err)
		return nil, 0, false
	}

	// 3. Parsear datos de Memcached
	var data cacheData
	if err := json.Unmarshal(memcachedItem.Value, &data); err != nil {
		log.Printf("Error unmarshaling cache data from Memcached: key=%s, error=%v", key, err)
		return nil, 0, false
	}

	// 4. Guardar en caché local para próximas consultas
	r.localCache.Set(key, &data, 5*time.Minute)
	log.Printf("Cache HIT (Memcached): key=%s, stored in local cache", key)

	return data.Properties, data.Total, true
}

// Set guarda datos en ambos niveles de caché
func (r *cacheRepository) Set(key string, properties []domain.Property, total int, ttl time.Duration) {
	data := &cacheData{
		Properties: properties,
		Total:      total,
	}

	// 1. Guardar en caché local con TTL de 5 minutos
	r.localCache.Set(key, data, 5*time.Minute)
	log.Printf("Cache SET (local): key=%s, ttl=5m", key)

	// 2. Serializar a JSON para Memcached
	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Printf("Error marshaling cache data for Memcached: key=%s, error=%v", key, err)
		return
	}

	// 3. Guardar en Memcached con TTL de 15 minutos
	// Convertir ttl a segundos (Memcached usa segundos)
	memcachedTTL := int32(15 * 60) // 15 minutos en segundos
	
	memcachedItem := &memcache.Item{
		Key:        key,
		Value:      jsonData,
		Expiration: memcachedTTL,
	}

	if err := r.memcachedClient.Set(memcachedItem); err != nil {
		log.Printf("Error setting cache in Memcached: key=%s, error=%v", key, err)
		return
	}

	log.Printf("Cache SET (Memcached): key=%s, ttl=15m", key)
}

// Delete elimina datos de ambos niveles de caché
func (r *cacheRepository) Delete(key string) {
	// 1. Eliminar de caché local
	r.localCache.Delete(key)
	log.Printf("Cache DELETE (local): key=%s", key)

	// 2. Eliminar de Memcached
	if err := r.memcachedClient.Delete(key); err != nil {
		if err == memcache.ErrCacheMiss {
			log.Printf("Cache DELETE (Memcached): key=%s (not found)", key)
			return
		}
		log.Printf("Error deleting from Memcached: key=%s, error=%v", key, err)
		return
	}

	log.Printf("Cache DELETE (Memcached): key=%s", key)
}


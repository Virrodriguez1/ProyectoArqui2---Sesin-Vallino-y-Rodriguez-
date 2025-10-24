# ProyectoArqui2---Sesin-Vallino-y-Rodriguez-


# 🏠 Airbnb Microservices

Sistema de reservas de alojamientos con arquitectura de microservicios.  
**Proyecto Final - Arquitectura de Software II - UCC**

---

## 🏗️ Arquitectura

### Backend (Go)
- **users-api** (8080): Gestión de usuarios, JWT, MySQL + GORM
- **properties-api** (8081): CRUD propiedades/reservas, MongoDB, RabbitMQ, concurrencia
- **search-api** (8082): Búsqueda con Solr, caché (CCache + Memcached), consumer RabbitMQ

### Frontend (React)
Login, Registro, Búsqueda, Detalles, Reserva, Mis Reservas, Admin

---

## 🚀 Instalación

```bash
# Clonar repositorio
git clone <url>
cd airbnb-microservices

# Configurar variables
cp .env.example .env

# Levantar servicios
docker-compose up --build
```

### URLs
- Frontend: http://localhost:3000
- users-api: http://localhost:8080
- properties-api: http://localhost:8081
- search-api: http://localhost:8082
- RabbitMQ: http://localhost:15672
- Solr: http://localhost:8983

---

## 📡 Endpoints

### users-api
```
POST /users          # Crear usuario
GET  /users/:id      # Obtener usuario
POST /users/login    # Login (JWT)
```

### properties-api
```
POST   /properties         # Crear propiedad
GET    /properties/:id     # Obtener propiedad
PUT    /properties/:id     # Actualizar
DELETE /properties/:id     # Eliminar
POST   /bookings           # Crear reserva
GET    /bookings/user/:id  # Reservas de usuario
```

### search-api
```
GET /search?query=...&page=1&size=10  # Búsqueda paginada
```

---

## 🛠️ Stack

**Backend:** Go, MySQL, MongoDB, RabbitMQ, Solr, Memcached  
**Frontend:** React  
**DevOps:** Docker, Docker Compose

---

## 🧪 Testing

```bash
cd users-api
go test ./services -v
```

---

## 👥 Equipo

- Desarrollador 1
- Desarrollador 2
- Desarrollador 3

---

## 📚 Recursos

- [Go Docs](https://go.dev/doc/)
- [GORM](https://gorm.io/docs/)
- [MongoDB Go Driver](https://www.mongodb.com/docs/drivers/go/current/)
- [RabbitMQ](https://www.rabbitmq.com/tutorials)
- [Solr](https://solr.apache.org/guide/)

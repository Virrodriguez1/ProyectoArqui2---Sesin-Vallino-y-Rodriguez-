package main

import (
	"fmt"
	"log"
	"os"
	"users-api/controllers"
	"users-api/domain"
	"users-api/middleware"
	"users-api/repositories"
	"users-api/services"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	// ============================================
	// 1. CONFIGURACIÓN - Leer variables de entorno
	// ============================================
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "3306")
	dbUser := getEnv("DB_USER", "spotly_user")
	dbPassword := getEnv("DB_PASSWORD", "spotly_password")
	dbName := getEnv("DB_NAME", "users_db")

	log.Println("🔧 Configuración cargada:")
	log.Printf("   - DB Host: %s:%s", dbHost, dbPort)
	log.Printf("   - DB Name: %s", dbName)

	// ============================================
	// 2. CONECTAR A MYSQL
	// ============================================
	// DSN = Data Source Name (string de conexión)
	// Formato: usuario:password@tcp(host:puerto)/base_de_datos?opciones
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		dbUser, dbPassword, dbHost, dbPort, dbName)

	log.Println("📡 Conectando a MySQL...")
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("❌ Failed to connect to database:", err)
	}
	log.Println("✅ Conexión a MySQL exitosa")

	// ============================================
	// 3. AUTO-MIGRAR LAS TABLAS
	// ============================================
	// GORM crea automáticamente la tabla "users" si no existe
	log.Println("🔄 Ejecutando migraciones...")
	err = db.AutoMigrate(&domain.User{})
	if err != nil {
		log.Fatal("❌ Failed to migrate database:", err)
	}
	log.Println("✅ Tablas creadas/actualizadas")

	// ============================================
	// 4. INICIALIZAR CAPAS (Patrón MVC)
	// ============================================
	log.Println("🏗️  Inicializando capas...")

	// Repository: acceso a datos
	userRepo := repositories.NewUserRepository(db)

	// Service: lógica de negocio
	userService := services.NewUserService(userRepo)

	// Controller: maneja HTTP
	userController := controllers.NewUserController(userService)

	log.Println("✅ Capas inicializadas")

	// ============================================
	// 5. CONFIGURAR GIN (Framework web)
	// ============================================
	// Gin es como Express en Node.js
	router := gin.Default()

	// CORS - Permitir requests desde el frontend
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// ============================================
	// 6. DEFINIR RUTAS (Endpoints)
	// ============================================
	log.Println("🛣️  Configurando rutas...")

	// Rutas PÚBLICAS (sin autenticación)
	router.GET("/health", userController.HealthCheck)
	router.POST("/users", userController.CreateUser)     // Registro
	router.POST("/users/login", userController.Login)    // Login
	router.GET("/users/:id", userController.GetUserByID) // Obtener usuario

	// Rutas PROTEGIDAS (requieren JWT - solo admin)
	// Importar middleware aquí si no está importado
	admin := router.Group("/admin")
	admin.Use(middleware.AuthMiddleware(), middleware.AdminMiddleware())
	{
		admin.GET("/users", userController.GetAllUsers)       // Listar todos
		admin.PUT("/users/:id", userController.UpdateUser)    // Actualizar
		admin.DELETE("/users/:id", userController.DeleteUser) // Eliminar
	}

	log.Println("✅ Rutas configuradas:")
	log.Println("   - GET  /health")
	log.Println("   - POST /users (registro)")
	log.Println("   - POST /users/login")
	log.Println("   - GET  /users/:id")
	log.Println("   - GET  /admin/users (admin)")
	log.Println("   - PUT  /admin/users/:id (admin)")
	log.Println("   - DELETE /admin/users/:id (admin)")

	// ============================================
	// 7. ARRANCAR EL SERVIDOR
	// ============================================
	port := getEnv("SERVER_PORT", "8080")

	log.Println("🚀 =======================================")
	log.Printf("🚀 Users API corriendo en puerto %s", port)
	log.Println("🚀 =======================================")

	if err := router.Run(":" + port); err != nil {
		log.Fatal("❌ Failed to start server:", err)
	}
}

// getEnv obtiene una variable de entorno o retorna un valor por defecto
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

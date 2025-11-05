package middleware

import (
	"net/http"
	"strings"
	"users-api/utils"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware valida el JWT token en cada request
// Si el token es válido, permite continuar
// Si no, devuelve error 401 (Unauthorized)
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Obtener el header "Authorization"
		authHeader := c.GetHeader("Authorization")

		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "authorization header required",
			})
			c.Abort() // Detiene la ejecución
			return
		}

		// Formato esperado: "Bearer <token>"
		// Ejemplo: "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "invalid authorization header format",
			})
			c.Abort()
			return
		}

		// Extraer el token
		tokenString := parts[1]

		// Validar el token
		claims, err := utils.ValidateToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "invalid or expired token",
			})
			c.Abort()
			return
		}

		// Guardar la info del usuario en el contexto
		// Así los endpoints pueden saber quién hizo la request
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("user_type", claims.UserType)

		c.Next() // Continúa con el endpoint
	}
}

// AdminMiddleware valida que el usuario sea admin
// Este middleware se usa DESPUÉS de AuthMiddleware
func AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		userType, exists := c.Get("user_type")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "user type not found",
			})
			c.Abort()
			return
		}

		if userType != "admin" {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "admin privileges required",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

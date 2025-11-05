package utils

import (
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Esta es la "llave secreta" para firmar los tokens
// En producción debe estar en variables de entorno
var jwtSecret = []byte(getJWTSecret())

// Claims es la estructura de los datos que guardamos EN el token
// Cuando el usuario hace login, le damos un token con esta info
type Claims struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	UserType string `json:"user_type"`
	jwt.RegisteredClaims
}

// getJWTSecret obtiene el secret desde variables de entorno
// Si no existe, usa uno por defecto (solo para desarrollo)
func getJWTSecret() string {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "default-secret-change-in-production"
	}
	return secret
}

// GenerateToken genera un nuevo JWT token para un usuario
// Se llama después del login exitoso
func GenerateToken(userID uint, username, userType string) (string, error) {
	// El token expira en 24 horas
	expirationTime := time.Now().Add(24 * time.Hour)

	// Creamos los "claims" (datos que va a tener el token)
	claims := &Claims{
		UserID:   userID,
		Username: username,
		UserType: userType,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	// Creamos el token y lo firmamos con nuestro secret
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// ValidateToken valida un JWT token y retorna los claims
// Se usa en el middleware para verificar que el usuario esté autenticado
func ValidateToken(tokenString string) (*Claims, error) {
	claims := &Claims{}

	// Parseamos el token y verificamos la firma
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}

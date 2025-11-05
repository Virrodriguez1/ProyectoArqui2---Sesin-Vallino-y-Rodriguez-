package utils

import (
	"golang.org/x/crypto/bcrypt"
)

// HashPassword hashea una contraseña usando bcrypt
// Recibe: "mipassword123"
// Devuelve: "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy"
func HashPassword(password string) (string, error) {
	// bcrypt.DefaultCost = 10 (nivel de seguridad)
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckPasswordHash verifica si una contraseña coincide con el hash
// Se usa en el login para verificar que la contraseña sea correcta
// Recibe: "mipassword123" y el hash guardado en la BD
// Devuelve: true si coincide, false si no
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

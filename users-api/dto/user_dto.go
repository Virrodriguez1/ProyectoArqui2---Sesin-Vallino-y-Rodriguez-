package dto

import "users-api/domain"

// CreateUserRequest representa el request para crear un usuario
// Esto es lo que el frontend te envía cuando alguien se registra
type CreateUserRequest struct {
	Username  string `json:"username" binding:"required,min=3,max=50"`
	Email     string `json:"email" binding:"required,email"`
	Password  string `json:"password" binding:"required,min=6"`
	FirstName string `json:"first_name" binding:"required"`
	LastName  string `json:"last_name" binding:"required"`
}

// LoginRequest representa el request para login
// El usuario puede loguearse con username O email
type LoginRequest struct {
	UsernameOrEmail string `json:"username_or_email" binding:"required"`
	Password        string `json:"password" binding:"required"`
}

// UpdateUserRequest representa el request para actualizar un usuario
// Todos los campos son opcionales
type UpdateUserRequest struct {
	Username  string `json:"username,omitempty"`
	Email     string `json:"email,omitempty" binding:"omitempty,email"`
	Password  string `json:"password,omitempty" binding:"omitempty,min=6"`
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
}

// LoginResponse representa la respuesta del login
// Devuelves el token JWT y los datos del usuario
type LoginResponse struct {
	Token string      `json:"token"`
	User  domain.User `json:"user"`
}

// UserResponse representa la respuesta con datos de usuario
// (Opcional, si querés una respuesta más limpia sin algunos campos)
type UserResponse struct {
	ID        uint   `json:"id"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	UserType  string `json:"user_type"`
}

// ErrorResponse representa una respuesta de error
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// SuccessResponse representa una respuesta exitosa
type SuccessResponse struct {
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

package controllers

import (
	"net/http"
	"strconv"
	"users-api/dto"
	"users-api/services"

	"github.com/gin-gonic/gin"
)

// UserController maneja los endpoints HTTP de usuarios
type UserController struct {
	service services.UserService
}

// NewUserController crea una nueva instancia del controlador
func NewUserController(service services.UserService) *UserController {
	return &UserController{service: service}
}

// CreateUser maneja POST /users
// Este endpoint se usa para REGISTRAR un nuevo usuario
func (ctrl *UserController) CreateUser(c *gin.Context) {
	// 1. Leer el JSON del body y parsearlo a CreateUserRequest
	var req dto.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Si el JSON es inválido o faltan campos, devolver error 400
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "validation_error",
			Message: err.Error(),
		})
		return
	}

	// 2. Llamar al servicio para crear el usuario
	user, err := ctrl.service.CreateUser(req)
	if err != nil {
		// Si hay error (username duplicado, etc), devolver 400
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "create_user_error",
			Message: err.Error(),
		})
		return
	}

	// 3. Devolver respuesta exitosa con el usuario creado
	// Status 201 = Created
	c.JSON(http.StatusCreated, dto.SuccessResponse{
		Message: "User created successfully",
		Data:    user,
	})
}

// GetUserByID maneja GET /users/:id
// Este endpoint obtiene un usuario por su ID
// Ejemplo: GET /users/5 -> obtiene el usuario con ID 5
func (ctrl *UserController) GetUserByID(c *gin.Context) {
	// 1. Obtener el parámetro "id" de la URL
	idParam := c.Param("id")

	// 2. Convertir el string a número
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid user ID",
		})
		return
	}

	// 3. Llamar al servicio para obtener el usuario
	user, err := ctrl.service.GetUserByID(uint(id))
	if err != nil {
		// Si no existe, devolver 404 (Not Found)
		c.JSON(http.StatusNotFound, dto.ErrorResponse{
			Error:   "user_not_found",
			Message: err.Error(),
		})
		return
	}

	// 4. Devolver el usuario encontrado
	c.JSON(http.StatusOK, user)
}

// Login maneja POST /users/login
// Este es el endpoint más importante: autentica al usuario
func (ctrl *UserController) Login(c *gin.Context) {
	// 1. Leer el JSON del body
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "validation_error",
			Message: err.Error(),
		})
		return
	}

	// 2. Llamar al servicio para hacer login
	// El servicio valida contraseña y genera el JWT
	response, err := ctrl.service.Login(req)
	if err != nil {
		// Si las credenciales son incorrectas, devolver 401 (Unauthorized)
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error:   "login_error",
			Message: err.Error(),
		})
		return
	}

	// 3. Devolver el token JWT y los datos del usuario
	c.JSON(http.StatusOK, response)
}

// HealthCheck maneja GET /health
// Endpoint simple para verificar que el servicio está corriendo
func (ctrl *UserController) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "users-api",
	})
}

// UpdateUser maneja PUT /users/:id
// Este endpoint actualiza un usuario existente
// Solo el admin o el propio usuario pueden actualizarse
func (ctrl *UserController) UpdateUser(c *gin.Context) {
	// 1. Obtener el ID de la URL
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid user ID",
		})
		return
	}

	// 2. Leer el JSON del body
	var req dto.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "validation_error",
			Message: err.Error(),
		})
		return
	}

	// 3. Llamar al servicio para actualizar
	user, err := ctrl.service.UpdateUser(uint(id), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "update_user_error",
			Message: err.Error(),
		})
		return
	}

	// 4. Devolver el usuario actualizado
	c.JSON(http.StatusOK, dto.SuccessResponse{
		Message: "User updated successfully",
		Data:    user,
	})
}

// DeleteUser maneja DELETE /users/:id
// Este endpoint elimina un usuario
// Solo el admin puede eliminar usuarios
func (ctrl *UserController) DeleteUser(c *gin.Context) {
	// 1. Obtener el ID de la URL
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid user ID",
		})
		return
	}

	// 2. Llamar al servicio para eliminar
	err = ctrl.service.DeleteUser(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, dto.ErrorResponse{
			Error:   "delete_user_error",
			Message: err.Error(),
		})
		return
	}

	// 3. Devolver confirmación
	c.JSON(http.StatusOK, dto.SuccessResponse{
		Message: "User deleted successfully",
	})
}

// GetAllUsers maneja GET /users
// Este endpoint lista todos los usuarios
// Solo accesible por administradores
func (ctrl *UserController) GetAllUsers(c *gin.Context) {
	// 1. Llamar al servicio para obtener todos los usuarios
	users, err := ctrl.service.GetAllUsers()
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "get_users_error",
			Message: err.Error(),
		})
		return
	}

	// 2. Devolver la lista de usuarios
	c.JSON(http.StatusOK, dto.SuccessResponse{
		Message: "Users retrieved successfully",
		Data:    users,
	})
}

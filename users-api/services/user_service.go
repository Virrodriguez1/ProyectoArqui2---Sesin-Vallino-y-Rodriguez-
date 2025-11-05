package services

import (
	"errors"
	"strings"
	"users-api/domain"
	"users-api/dto"
	"users-api/repositories"
	"users-api/utils"
)

// UserService define la interfaz del servicio
type UserService interface {
	CreateUser(req dto.CreateUserRequest) (*domain.User, error)
	GetUserByID(id uint) (*domain.User, error)
	Login(req dto.LoginRequest) (*dto.LoginResponse, error)
}

// userService es la implementación real del servicio
// Tiene un repositorio para acceder a la base de datos
type userService struct {
	repo repositories.UserRepository
}

// NewUserService crea una nueva instancia del servicio
func NewUserService(repo repositories.UserRepository) UserService {
	return &userService{repo: repo}
}

// CreateUser crea un nuevo usuario
// Aquí va toda la lógica: validaciones, hashear password, etc.
func (s *userService) CreateUser(req dto.CreateUserRequest) (*domain.User, error) {
	// 1. Verificar si el username ya existe
	existingUser, _ := s.repo.GetByUsername(req.Username)
	if existingUser != nil {
		return nil, errors.New("username already exists")
	}

	// 2. Verificar si el email ya existe
	existingUser, _ = s.repo.GetByEmail(req.Email)
	if existingUser != nil {
		return nil, errors.New("email already exists")
	}

	// 3. Hashear la contraseña
	// NUNCA guardamos contraseñas en texto plano
	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		return nil, errors.New("error hashing password")
	}

	// 4. Crear el objeto User
	user := &domain.User{
		Username:  req.Username,
		Email:     req.Email,
		Password:  hashedPassword, // Guardamos el hash, no la contraseña
		FirstName: req.FirstName,
		LastName:  req.LastName,
		UserType:  domain.UserTypeNormal, // Por defecto es usuario normal
	}

	// 5. Guardar en la base de datos
	err = s.repo.Create(user)
	if err != nil {
		return nil, err
	}

	return user, nil
}

// GetUserByID obtiene un usuario por su ID
// Esta función es simple, solo delega al repositorio
func (s *userService) GetUserByID(id uint) (*domain.User, error) {
	return s.repo.GetByID(id)
}

// Login autentica un usuario y genera un token JWT
// Esta es la función más importante del servicio
func (s *userService) Login(req dto.LoginRequest) (*dto.LoginResponse, error) {
	var user *domain.User
	var err error

	// 1. Determinar si el usuario está intentando loguearse con username o email
	// Si contiene "@" asumimos que es email
	if strings.Contains(req.UsernameOrEmail, "@") {
		user, err = s.repo.GetByEmail(req.UsernameOrEmail)
	} else {
		user, err = s.repo.GetByUsername(req.UsernameOrEmail)
	}

	// 2. Si no encontramos el usuario, devolvemos error genérico
	// (Por seguridad, no decimos si el username existe o no)
	if err != nil {
		return nil, errors.New("invalid credentials")
	}

	// 3. Verificar que la contraseña sea correcta
	// Comparamos el hash guardado con la contraseña que envió
	if !utils.CheckPasswordHash(req.Password, user.Password) {
		return nil, errors.New("invalid credentials")
	}

	// 4. Generar el token JWT
	// Este token contiene: user_id, username, user_type
	token, err := utils.GenerateToken(user.ID, user.Username, string(user.UserType))
	if err != nil {
		return nil, errors.New("error generating token")
	}

	// 5. Devolver el token y los datos del usuario
	return &dto.LoginResponse{
		Token: token,
		User:  *user,
	}, nil

	// UpdateUser actualiza los datos de un usuario existente
	func (s *userService) UpdateUser(id uint, req dto.UpdateUserRequest) (*domain.User, error) {
		// 1. Verificar que el usuario existe
		user, err := s.repo.GetByID(id)
		if err != nil {
			return nil, errors.New("user not found")
		}

		// 2. Si se proporciona un nuevo username, verificar que no esté en uso
		if req.Username != "" && req.Username != user.Username {
			existingUser, _ := s.repo.GetByUsername(req.Username)
			if existingUser != nil {
				return nil, errors.New("username already exists")
			}
			user.Username = req.Username
		}

		// 3. Si se proporciona un nuevo email, verificar que no esté en uso
		if req.Email != "" && req.Email != user.Email {
			existingUser, _ := s.repo.GetByEmail(req.Email)
			if existingUser != nil {
				return nil, errors.New("email already exists")
			}
			user.Email = req.Email
		}

		// 4. Actualizar otros campos si se proporcionan
		if req.FirstName != "" {
			user.FirstName = req.FirstName
		}

		if req.LastName != "" {
			user.LastName = req.LastName
		}

		// 5. Si se proporciona una nueva contraseña, hashearla
		if req.Password != "" {
			hashedPassword, err := utils.HashPassword(req.Password)
			if err != nil {
				return nil, errors.New("error hashing password")
			}
			user.Password = hashedPassword
		}

		// 6. Guardar los cambios en la base de datos
		err = s.repo.Update(user)
		if err != nil {
			return nil, err
		}

		return user, nil
	}

	// DeleteUser elimina un usuario por su ID
	func (s *userService) DeleteUser(id uint) error {
		// 1. Verificar que el usuario existe
		_, err := s.repo.GetByID(id)
		if err != nil {
		return errors.New("user not found")
	}

		// 2. Eliminar el usuario
		return s.repo.Delete(id)
	}

	// GetAllUsers obtiene todos los usuarios del sistema
	// Solo accesible por administradores
	func (s *userService) GetAllUsers() ([]domain.User, error) {
		return s.repo.GetAll()
	}

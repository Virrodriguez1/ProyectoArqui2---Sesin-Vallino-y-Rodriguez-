package repositories

import (
	"errors"
	"users-api/domain"

	"gorm.io/gorm"
)

// UserRepository define la interfaz del repositorio
// Es como un "contrato" que dice qué operaciones debe tener
type UserRepository interface {
	Create(user *domain.User) error
	GetByID(id uint) (*domain.User, error)
	GetByUsername(username string) (*domain.User, error)
	GetByEmail(email string) (*domain.User, error)
	Update(user *domain.User) error
	Delete(id uint) error
	GetAll() ([]domain.User, error)
}

// userRepository es la implementación real del repositorio
// Tiene una conexión a la base de datos (db)
type userRepository struct {
	db *gorm.DB
}

// NewUserRepository crea una nueva instancia del repositorio
// Recibe la conexión a la base de datos
func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

// Create inserta un nuevo usuario en la base de datos
// GORM automáticamente hace el INSERT
func (r *userRepository) Create(user *domain.User) error {
	return r.db.Create(user).Error
}

// GetByID busca un usuario por su ID
// Ejemplo: GetByID(1) -> SELECT * FROM users WHERE id = 1
func (r *userRepository) GetByID(id uint) (*domain.User, error) {
	var user domain.User
	err := r.db.First(&user, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	return &user, nil
}

// GetByUsername busca un usuario por su username
// Se usa en el login cuando el usuario pone su username
func (r *userRepository) GetByUsername(username string) (*domain.User, error) {
	var user domain.User
	err := r.db.Where("username = ?", username).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	return &user, nil
}

// GetByEmail busca un usuario por su email
// Se usa en el login cuando el usuario pone su email
func (r *userRepository) GetByEmail(email string) (*domain.User, error) {
	var user domain.User
	err := r.db.Where("email = ?", email).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	return &user, nil
}

// Update actualiza un usuario existente
// GORM hace UPDATE de todos los campos
func (r *userRepository) Update(user *domain.User) error {
	return r.db.Save(user).Error
}

// Delete elimina un usuario por su ID
// GORM hace DELETE FROM users WHERE id = ?
func (r *userRepository) Delete(id uint) error {
	return r.db.Delete(&domain.User{}, id).Error
}

// GetAll obtiene todos los usuarios
// GORM hace SELECT * FROM users
func (r *userRepository) GetAll() ([]domain.User, error) {
	var users []domain.User
	err := r.db.Find(&users).Error
	return users, err
}

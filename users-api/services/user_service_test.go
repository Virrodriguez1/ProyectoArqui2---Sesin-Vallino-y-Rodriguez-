package services

import (
	"errors"
	"testing"
	"users-api/domain"
	"users-api/dto"
)

// ============================================
// MOCK del repositorio para los tests
// ============================================
type mockUserRepository struct {
	users map[uint]*domain.User
}

func newMockUserRepository() *mockUserRepository {
	return &mockUserRepository{
		users: make(map[uint]*domain.User),
	}
}

func (m *mockUserRepository) Create(user *domain.User) error {
	// Simular auto-increment del ID
	user.ID = uint(len(m.users) + 1)
	m.users[user.ID] = user
	return nil
}

func (m *mockUserRepository) GetByID(id uint) (*domain.User, error) {
	user, exists := m.users[id]
	if !exists {
		return nil, errors.New("user not found")
	}
	return user, nil
}

func (m *mockUserRepository) GetByUsername(username string) (*domain.User, error) {
	for _, user := range m.users {
		if user.Username == username {
			return user, nil
		}
	}
	return nil, errors.New("user not found")
}

func (m *mockUserRepository) GetByEmail(email string) (*domain.User, error) {
	for _, user := range m.users {
		if user.Email == email {
			return user, nil
		}
	}
	return nil, errors.New("user not found")
}

func (m *mockUserRepository) Update(user *domain.User) error {
	if _, exists := m.users[user.ID]; !exists {
		return errors.New("user not found")
	}
	m.users[user.ID] = user
	return nil
}

func (m *mockUserRepository) Delete(id uint) error {
	if _, exists := m.users[id]; !exists {
		return errors.New("user not found")
	}
	delete(m.users, id)
	return nil
}

// ============================================
// TESTS
// ============================================

// Test: Crear usuario exitosamente
func TestCreateUser_Success(t *testing.T) {
	repo := newMockUserRepository()
	service := NewUserService(repo)

	req := dto.CreateUserRequest{
		Username:  "testuser",
		Email:     "test@example.com",
		Password:  "password123",
		FirstName: "Test",
		LastName:  "User",
	}

	user, err := service.CreateUser(req)

	// Verificaciones
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if user == nil {
		t.Fatal("Expected user, got nil")
	}

	if user.Username != req.Username {
		t.Errorf("Expected username %s, got %s", req.Username, user.Username)
	}

	if user.Email != req.Email {
		t.Errorf("Expected email %s, got %s", req.Email, user.Email)
	}

	if user.UserType != domain.UserTypeNormal {
		t.Errorf("Expected user type %s, got %s", domain.UserTypeNormal, user.UserType)
	}

	// Verificar que la contraseña fue hasheada (no es la original)
	if user.Password == req.Password {
		t.Error("Password should be hashed, not plain text")
	}
}

// Test: Error al crear usuario con username duplicado
func TestCreateUser_DuplicateUsername(t *testing.T) {
	repo := newMockUserRepository()
	service := NewUserService(repo)

	// Crear primer usuario
	req1 := dto.CreateUserRequest{
		Username:  "testuser",
		Email:     "test1@example.com",
		Password:  "password123",
		FirstName: "Test",
		LastName:  "User",
	}
	service.CreateUser(req1)

	// Intentar crear segundo usuario con mismo username
	req2 := dto.CreateUserRequest{
		Username:  "testuser", // Username duplicado
		Email:     "test2@example.com",
		Password:  "password123",
		FirstName: "Test",
		LastName:  "User",
	}

	user, err := service.CreateUser(req2)

	// Verificaciones
	if err == nil {
		t.Error("Expected error for duplicate username, got nil")
	}

	if user != nil {
		t.Error("Expected nil user, got user")
	}

	if err.Error() != "username already exists" {
		t.Errorf("Expected 'username already exists' error, got %v", err)
	}
}

// Test: Error al crear usuario con email duplicado
func TestCreateUser_DuplicateEmail(t *testing.T) {
	repo := newMockUserRepository()
	service := NewUserService(repo)

	// Crear primer usuario
	req1 := dto.CreateUserRequest{
		Username:  "testuser1",
		Email:     "test@example.com",
		Password:  "password123",
		FirstName: "Test",
		LastName:  "User",
	}
	service.CreateUser(req1)

	// Intentar crear segundo usuario con mismo email
	req2 := dto.CreateUserRequest{
		Username:  "testuser2",
		Email:     "test@example.com", // Email duplicado
		Password:  "password123",
		FirstName: "Test",
		LastName:  "User",
	}

	user, err := service.CreateUser(req2)

	// Verificaciones
	if err == nil {
		t.Error("Expected error for duplicate email, got nil")
	}

	if user != nil {
		t.Error("Expected nil user, got user")
	}

	if err.Error() != "email already exists" {
		t.Errorf("Expected 'email already exists' error, got %v", err)
	}
}

// Test: Login exitoso con username
func TestLogin_SuccessWithUsername(t *testing.T) {
	repo := newMockUserRepository()
	service := NewUserService(repo)

	// Crear usuario
	createReq := dto.CreateUserRequest{
		Username:  "testuser",
		Email:     "test@example.com",
		Password:  "password123",
		FirstName: "Test",
		LastName:  "User",
	}
	service.CreateUser(createReq)

	// Intentar login
	loginReq := dto.LoginRequest{
		UsernameOrEmail: "testuser",
		Password:        "password123",
	}

	response, err := service.Login(loginReq)

	// Verificaciones
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if response == nil {
		t.Fatal("Expected login response, got nil")
	}

	if response.Token == "" {
		t.Error("Expected JWT token, got empty string")
	}

	if response.User.Username != "testuser" {
		t.Errorf("Expected username testuser, got %s", response.User.Username)
	}
}

// Test: Login exitoso con email
func TestLogin_SuccessWithEmail(t *testing.T) {
	repo := newMockUserRepository()
	service := NewUserService(repo)

	// Crear usuario
	createReq := dto.CreateUserRequest{
		Username:  "testuser",
		Email:     "test@example.com",
		Password:  "password123",
		FirstName: "Test",
		LastName:  "User",
	}
	service.CreateUser(createReq)

	// Intentar login con email
	loginReq := dto.LoginRequest{
		UsernameOrEmail: "test@example.com",
		Password:        "password123",
	}

	response, err := service.Login(loginReq)

	// Verificaciones
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if response == nil {
		t.Fatal("Expected login response, got nil")
	}

	if response.Token == "" {
		t.Error("Expected JWT token, got empty string")
	}
}

// Test: Login fallido - usuario no existe
func TestLogin_UserNotFound(t *testing.T) {
	repo := newMockUserRepository()
	service := NewUserService(repo)

	loginReq := dto.LoginRequest{
		UsernameOrEmail: "nonexistent",
		Password:        "password123",
	}

	response, err := service.Login(loginReq)

	// Verificaciones
	if err == nil {
		t.Error("Expected error for non-existent user, got nil")
	}

	if response != nil {
		t.Error("Expected nil response, got response")
	}

	if err.Error() != "invalid credentials" {
		t.Errorf("Expected 'invalid credentials' error, got %v", err)
	}
}

// Test: Login fallido - contraseña incorrecta
func TestLogin_WrongPassword(t *testing.T) {
	repo := newMockUserRepository()
	service := NewUserService(repo)

	// Crear usuario
	createReq := dto.CreateUserRequest{
		Username:  "testuser",
		Email:     "test@example.com",
		Password:  "password123",
		FirstName: "Test",
		LastName:  "User",
	}
	service.CreateUser(createReq)

	// Intentar login con contraseña incorrecta
	loginReq := dto.LoginRequest{
		UsernameOrEmail: "testuser",
		Password:        "wrongpassword",
	}

	response, err := service.Login(loginReq)

	// Verificaciones
	if err == nil {
		t.Error("Expected error for wrong password, got nil")
	}

	if response != nil {
		t.Error("Expected nil response, got response")
	}

	if err.Error() != "invalid credentials" {
		t.Errorf("Expected 'invalid credentials' error, got %v", err)
	}
}

// Test: Obtener usuario por ID exitosamente
func TestGetUserByID_Success(t *testing.T) {
	repo := newMockUserRepository()
	service := NewUserService(repo)

	// Crear usuario
	createReq := dto.CreateUserRequest{
		Username:  "testuser",
		Email:     "test@example.com",
		Password:  "password123",
		FirstName: "Test",
		LastName:  "User",
	}
	createdUser, _ := service.CreateUser(createReq)

	// Obtener usuario por ID
	user, err := service.GetUserByID(createdUser.ID)

	// Verificaciones
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if user == nil {
		t.Fatal("Expected user, got nil")
	}

	if user.ID != createdUser.ID {
		t.Errorf("Expected ID %d, got %d", createdUser.ID, user.ID)
	}
}

// Test: Error al obtener usuario que no existe
func TestGetUserByID_NotFound(t *testing.T) {
	repo := newMockUserRepository()
	service := NewUserService(repo)

	// Intentar obtener usuario con ID inexistente
	user, err := service.GetUserByID(999)

	// Verificaciones
	if err == nil {
		t.Error("Expected error for non-existent user, got nil")
	}

	if user != nil {
		t.Error("Expected nil user, got user")
	}
}

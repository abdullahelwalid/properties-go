package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"golang-test/api/handler"
	"golang-test/models"
	"golang-test/utils"
)

// MockDB is a mock implementation of the database interface
type MockDB struct {
	mock.Mock
}

// Model mocks the gorm DB Model method
func (m *MockDB) Model(value interface{}) *MockDB {
	m.Called(value)
	return m
}

// Where mocks the gorm DB Where method
func (m *MockDB) Where(query interface{}, args ...interface{}) *MockDB {
	m.Called(query, args[0])
	return m
}

// First mocks the gorm DB First method
func (m *MockDB) First(dest interface{}, conds ...interface{}) *gorm.DB {
	args := m.Called(dest)
	// This simulates loading data into the destination struct
	if user, ok := dest.(*models.User); ok && args.Int(0) > 0 {
		user.ID = uint(args.Int(0))
		user.Email = "test@example.com"
		user.Name = "Test User"
		user.Password = "$2a$10$abcdefghijklmnopqrstuvwxyz" // Mocked hashed password
		user.RoleID = 1
	}
	return &gorm.DB{RowsAffected: int64(args.Int(0))}
}

var secretKey = []byte("your-secret-key")

func TestLoginSuccess(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	mockDB := new(MockDB)
	
	// Configure mock expectations
	mockDB.On("Model", mock.AnythingOfType("models.User")).Return(mockDB)
	mockDB.On("Where", "email = ?", "test@example.com").Return(mockDB)
	mockDB.On("First", mock.AnythingOfType("*models.User")).Return(1) // User found
	
	// Mock password verification
	origVerifyPassword := utils.VerifyPassword
	defer func() { utils.VerifyPassword = origVerifyPassword }()
	utils.VerifyPassword = func(password, hashedPassword string) bool {
		return true // Mock successful password verification
	}
	
	// Create handler with mock DB
	authHandler := &handlers.AuthHandler{DB: mockDB}
	
	// Create test HTTP request
	reqBody := map[string]string{
		"email":    "test@example.com",
		"password": "password123",
	}
	jsonBody, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	
	// Setup router and handler
	router := gin.New()
	router.POST("/api/auth/login", authHandler.Login)
	
	// Execute request
	router.ServeHTTP(resp, req)
	
	// Assertions
	assert.Equal(t, http.StatusOK, resp.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(resp.Body.Bytes(), &response)
	require.NoError(t, err)
	
	// Check response fields
	assert.NotEmpty(t, response["token"])
	assert.Equal(t, float64(1), response["id"])
	assert.Equal(t, "Test User", response["name"])
	assert.Equal(t, "test@example.com", response["email"])
	
	// Verify token is valid and contains expected claims
	tokenString := response["token"].(string)
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return secretKey, nil
	})
	require.NoError(t, err)
	
	claims := token.Claims.(jwt.MapClaims)
	assert.Equal(t, float64(1), claims["userId"])
	assert.Equal(t, float64(1), claims["roleId"])
	assert.NotEmpty(t, claims["exp"])
	
	// Verify mock expectations were met
	mockDB.AssertExpectations(t)
}

func TestLoginInvalidJSON(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	mockDB := new(MockDB)
	authHandler := &handler.AuthHandler{DB: mockDB}
	
	// Create test HTTP request with invalid JSON
	req, _ := http.NewRequest("POST", "/api/auth/login", bytes.NewBuffer([]byte(`{invalid json}`)))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	
	// Setup router and handler
	router := gin.New()
	router.POST("/api/auth/login", authHandler.Login)
	
	// Execute request
	router.ServeHTTP(resp, req)
	
	// Assertions
	assert.Equal(t, http.StatusBadRequest, resp.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(resp.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Equal(t, "Bad request, invalid request body", response["error"])
}

func TestLoginUserNotFound(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	mockDB := new(MockDB)
	
	// Configure mock expectations
	mockDB.On("Model", mock.AnythingOfType("models.User")).Return(mockDB)
	mockDB.On("Where", "email = ?", "nonexistent@example.com").Return(mockDB)
	mockDB.On("First", mock.AnythingOfType("*models.User")).Return(0) // User not found
	
	authHandler := &handler.AuthHandler{DB: mockDB}
	
	// Create test HTTP request
	reqBody := map[string]string{
		"email":    "nonexistent@example.com",
		"password": "password123",
	}
	jsonBody, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	
	// Setup router and handler
	router := gin.New()
	router.POST("/api/auth/login", authHandler.Login)
	
	// Execute request
	router.ServeHTTP(resp, req)
	
	// Assertions
	assert.Equal(t, http.StatusUnauthorized, resp.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(resp.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Equal(t, "Email or password is invalid", response["error"])
	
	// Verify mock expectations were met
	mockDB.AssertExpectations(t)
}

func TestLoginInvalidPassword(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	mockDB := new(MockDB)
	
	// Configure mock expectations
	mockDB.On("Model", mock.AnythingOfType("models.User")).Return(mockDB)
	mockDB.On("Where", "email = ?", "test@example.com").Return(mockDB)
	mockDB.On("First", mock.AnythingOfType("*models.User")).Return(1) // User found
	
	// Mock password verification
	origVerifyPassword := utils.VerifyPassword
	defer func() { utils.VerifyPassword = origVerifyPassword }()
	utils.VerifyPassword = func(password, hashedPassword string) bool {
		return false // Mock failed password verification
	}
	
	authHandler := &handlers.AuthHandler{DB: mockDB}
	
	// Create test HTTP request
	reqBody := map[string]string{
		"email":    "test@example.com",
		"password": "wrong_password",
	}
	jsonBody, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	
	// Setup router and handler
	router := gin.New()
	router.POST("/api/auth/login", authHandler.Login)
	
	// Execute request
	router.ServeHTTP(resp, req)
	
	// Assertions
	assert.Equal(t, http.StatusUnauthorized, resp.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(resp.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Equal(t, "Email or password is invalid", response["error"])
	
	// Verify mock expectations were met
	mockDB.AssertExpectations(t)
}

func TestLoginJWTError(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	mockDB := new(MockDB)
	
	// Configure mock expectations
	mockDB.On("Model", mock.AnythingOfType("models.User")).Return(mockDB)
	mockDB.On("Where", "email = ?", "test@example.com").Return(mockDB)
	mockDB.On("First", mock.AnythingOfType("*models.User")).Return(1) // User found
	
	// Mock password verification
	origVerifyPassword := utils.VerifyPassword
	defer func() { utils.VerifyPassword = origVerifyPassword }()
	utils.VerifyPassword = func(password, hashedPassword string) bool {
		return true // Mock successful password verification
	}
	
	// Replace the JWT signing function to simulate an error
	origSignedString := jwt.SignedString
	defer func() {
		// This is a bit tricky since we can't easily mock jwt.Token.SignedString
		// In a real test, you might need to use a different approach or a test-specific package
		// This is just to illustrate the concept
	}()
	
	// Mock the secretKey to force a signing error
	// For testing purposes - in real code, you'd use a different approach
	tempSecretKey := secretKey
	secretKey = nil // This will cause the SignedString method to fail
	defer func() {
		secretKey = tempSecretKey // Restore the original value
	}()
	
	authHandler := &handlers.AuthHandler{DB: mockDB}
	
	// Create test HTTP request
	reqBody := map[string]string{
		"email":    "test@example.com",
		"password": "password123",
	}
	jsonBody, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	
	// Setup router and handler
	router := gin.New()
	router.POST("/api/auth/login", authHandler.Login)
	
	// Execute request
	router.ServeHTTP(resp, req)
	
	// Assertions
	assert.Equal(t, http.StatusInternalServerError, resp.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(resp.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Equal(t, "An error has occurred while generating auth creds", response["error"])
	
	// Verify mock expectations were met
	mockDB.AssertExpectations(t)
}

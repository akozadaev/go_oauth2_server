package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"go_oauth2_server/internal/config"
	"go_oauth2_server/internal/models"
	"go_oauth2_server/internal/storage"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// getTestDB возвращает подключение к тестовой базе данных
func getTestDB(t *testing.T) *sql.DB {
	// Проверяем переменную окружения для тестовой БД
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		// Используем локальную БД для тестов
		dbURL = "postgres://test_user:test_password@localhost:5432/test_db?sslmode=disable"
	}

	db, err := sql.Open("postgres", dbURL)
	require.NoError(t, err)

	// Ждем пока база данных будет готова
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for i := 0; i < 30; i++ {
		if err := db.PingContext(ctx); err == nil {
			break
		}
		time.Sleep(1 * time.Second)
	}

	// Создаем таблицы
	err = createTestTables(db)
	require.NoError(t, err)

	return db
}

func createTestTables(db *sql.DB) error {
	queries := []string{
		`CREATE EXTENSION IF NOT EXISTS "uuid-ossp"`,
		`CREATE TABLE IF NOT EXISTS clients (
			id VARCHAR(255) PRIMARY KEY,
			secret VARCHAR(255) NOT NULL,
			domain VARCHAR(255),
			user_id VARCHAR(255),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS users (
			id VARCHAR(255) PRIMARY KEY,
			username VARCHAR(255) UNIQUE NOT NULL,
			password VARCHAR(255) NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS tokens (
			id VARCHAR(255) PRIMARY KEY,
			client_id VARCHAR(255) NOT NULL,
			user_id VARCHAR(255),
			redirect_uri VARCHAR(255),
			scope VARCHAR(255),
			code VARCHAR(255),
			code_created_at TIMESTAMP,
			code_expires_in INTEGER,
			access VARCHAR(255),
			access_created_at TIMESTAMP,
			access_expires_in INTEGER,
			refresh VARCHAR(255),
			refresh_created_at TIMESTAMP,
			refresh_expires_in INTEGER,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS oauth2_tokens (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			access_token VARCHAR(512) UNIQUE NOT NULL,
			refresh_token VARCHAR(512),
			client_id VARCHAR(255) NOT NULL,
			user_id VARCHAR(255),
			scope TEXT,
			access_expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
			refresh_expires_at TIMESTAMP WITH TIME ZONE,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,
	}

	for _, query := range queries {
		if _, err := db.Exec(query); err != nil {
			// Игнорируем ошибки если расширение уже существует
			if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
				continue
			}
			return fmt.Errorf("failed to create table: %w", err)
		}
	}

	return nil
}

func cleanupTestDB(db *sql.DB) error {
	queries := []string{
		"DELETE FROM oauth2_tokens",
		"DELETE FROM tokens",
		"DELETE FROM clients",
		"DELETE FROM users",
	}

	for _, query := range queries {
		if _, err := db.Exec(query); err != nil {
			// Игнорируем ошибки если таблица не существует
			if !strings.Contains(err.Error(), "does not exist") {
				return fmt.Errorf("failed to cleanup table: %w", err)
			}
		}
	}

	return nil
}

func setupTestHandler(t *testing.T) *Handler {
	db := getTestDB(t)

	// Очищаем данные перед тестом
	cleanupTestDB(db)

	store := storage.NewPostgresStore(db)
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg := &config.Config{
		Port:              "8080",
		JWTSecret:         "test-secret",
		TokenExpiration:   60 * time.Minute,
		RefreshExpiration: 168 * time.Hour,
		LogLevel:          "info",
	}

	handler := New(store, logger, cfg)
	return handler
}

func TestHandler_Health(t *testing.T) {
	handler := setupTestHandler(t)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	handler.Health(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "healthy", response["status"])
}

func TestHandler_RegisterUser(t *testing.T) {
	handler := setupTestHandler(t)

	userData := models.User{
		ID:        "test-user",
		Username:  "testuser",
		Password:  "testpassword",
		CreatedAt: time.Now(),
	}

	jsonData, err := json.Marshal(userData)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/users", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.RegisterUser(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	// Проверяем что ответ содержит user_id
	assert.Contains(t, response, "user_id")
}

func TestHandler_RegisterClient(t *testing.T) {
	handler := setupTestHandler(t)

	clientData := models.Client{
		ID:        "test-client",
		Secret:    "test-secret",
		Domain:    "http://localhost:3000",
		UserID:    "test-user",
		CreatedAt: time.Now(),
	}

	jsonData, err := json.Marshal(clientData)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/clients", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.RegisterClient(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	// Проверяем что ответ содержит client_id
	assert.Contains(t, response, "client_id")
}

func TestHandler_Integration(t *testing.T) {
	handler := setupTestHandler(t)

	// Создаем пользователя
	userData := models.User{
		ID:        "test-user",
		Username:  "testuser",
		Password:  "testpassword",
		CreatedAt: time.Now(),
	}

	userJSON, _ := json.Marshal(userData)
	userReq := httptest.NewRequest("POST", "/users", bytes.NewBuffer(userJSON))
	userReq.Header.Set("Content-Type", "application/json")
	userW := httptest.NewRecorder()
	handler.RegisterUser(userW, userReq)
	assert.Equal(t, http.StatusCreated, userW.Code)

	// Создаем клиента
	clientData := models.Client{
		ID:        "test-client",
		Secret:    "test-secret",
		Domain:    "http://localhost:3000",
		UserID:    userData.ID,
		CreatedAt: time.Now(),
	}

	clientJSON, _ := json.Marshal(clientData)
	clientReq := httptest.NewRequest("POST", "/clients", bytes.NewBuffer(clientJSON))
	clientReq.Header.Set("Content-Type", "application/json")
	clientW := httptest.NewRecorder()
	handler.RegisterClient(clientW, clientReq)
	assert.Equal(t, http.StatusCreated, clientW.Code)

	// Проверяем health endpoint
	healthReq := httptest.NewRequest("GET", "/health", nil)
	healthW := httptest.NewRecorder()
	handler.Health(healthW, healthReq)
	assert.Equal(t, http.StatusOK, healthW.Code)
}

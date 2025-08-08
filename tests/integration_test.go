package tests

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
	"go_oauth2_server/internal/handlers"
	"go_oauth2_server/internal/models"
	"go_oauth2_server/internal/storage"

	"github.com/go-chi/chi/v5"
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

func setupTestServer(t *testing.T) *httptest.Server {
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

	handler := handlers.New(store, logger, cfg)

	router := chi.NewRouter()
	router.HandleFunc("/authorize", handler.Authorize)
	router.HandleFunc("/token", handler.Token)
	router.HandleFunc("/introspect", handler.Introspect)
	router.HandleFunc("/clients", handler.RegisterClient)
	router.HandleFunc("/health", handler.Health)
	router.HandleFunc("/users", handler.RegisterUser)

	server := httptest.NewServer(router)
	return server
}

func TestOAuth2Flow_Integration(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	// Шаг 1: Создаем пользователя
	userData := models.User{
		ID:        "test-user-oauth",
		Username:  "testuser-oauth",
		Password:  "testpassword",
		CreatedAt: time.Now(),
	}

	userJSON, _ := json.Marshal(userData)
	userResp, err := http.Post(server.URL+"/users", "application/json", bytes.NewBuffer(userJSON))
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, userResp.StatusCode)

	// Шаг 2: Создаем клиента
	clientData := models.Client{
		ID:        "test-client-oauth",
		Secret:    "test-secret",
		Domain:    "http://localhost:3000",
		UserID:    userData.ID,
		CreatedAt: time.Now(),
	}

	clientJSON, _ := json.Marshal(clientData)
	clientResp, err := http.Post(server.URL+"/clients", "application/json", bytes.NewBuffer(clientJSON))
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, clientResp.StatusCode)

	// Шаг 3: Получаем код авторизации
	authorizeData := models.AuthorizeRequest{
		ResponseType: "code",
		ClientID:     clientData.ID,
		RedirectURI:  clientData.Domain,
		Scope:        "read",
		State:        "test-state",
		Username:     userData.Username,
		Password:     userData.Password,
	}

	authJSON, _ := json.Marshal(authorizeData)
	authResp, err := http.Post(server.URL+"/authorize", "application/json", bytes.NewBuffer(authJSON))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, authResp.StatusCode)

	var authResponse map[string]interface{}
	err = json.NewDecoder(authResp.Body).Decode(&authResponse)
	require.NoError(t, err)
	assert.Contains(t, authResponse, "code")

	code := authResponse["code"].(string)
	assert.NotEmpty(t, code)

	// Шаг 4: Получаем access token
	tokenData := models.TokenRequest{
		GrantType:    "authorization_code",
		Code:         code,
		RedirectURI:  clientData.Domain,
		ClientID:     clientData.ID,
		ClientSecret: clientData.Secret,
	}

	tokenJSON, _ := json.Marshal(tokenData)
	tokenResp, err := http.Post(server.URL+"/token", "application/json", bytes.NewBuffer(tokenJSON))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, tokenResp.StatusCode)

	var tokenResponse map[string]interface{}
	err = json.NewDecoder(tokenResp.Body).Decode(&tokenResponse)
	require.NoError(t, err)
	assert.Contains(t, tokenResponse, "access_token")
	assert.Contains(t, tokenResponse, "token_type")
	assert.Contains(t, tokenResponse, "expires_in")

	accessToken := tokenResponse["access_token"].(string)
	assert.NotEmpty(t, accessToken)

	// Шаг 5: Интроспекция токена
	introspectData := models.IntrospectRequest{
		Token: accessToken,
	}

	introspectJSON, _ := json.Marshal(introspectData)
	introspectResp, err := http.Post(server.URL+"/introspect", "application/json", bytes.NewBuffer(introspectJSON))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, introspectResp.StatusCode)

	var introspectResponse map[string]interface{}
	err = json.NewDecoder(introspectResp.Body).Decode(&introspectResponse)
	require.NoError(t, err)
	assert.Equal(t, true, introspectResponse["active"])
}

func TestHealthCheck_Integration(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	resp, err := http.Get(server.URL + "/health")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, "healthy", response["status"])
}

func TestUserRegistration_Integration(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	userData := models.User{
		ID:        "test-user",
		Username:  "testuser",
		Password:  "testpassword",
		CreatedAt: time.Now(),
	}

	userJSON, _ := json.Marshal(userData)
	resp, err := http.Post(server.URL+"/users", "application/json", bytes.NewBuffer(userJSON))
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)
	assert.Contains(t, response, "user_id")
}

func TestClientRegistration_Integration(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	clientData := models.Client{
		ID:        "test-client",
		Secret:    "test-secret",
		Domain:    "http://localhost:3000",
		UserID:    "test-user",
		CreatedAt: time.Now(),
	}

	clientJSON, _ := json.Marshal(clientData)
	resp, err := http.Post(server.URL+"/clients", "application/json", bytes.NewBuffer(clientJSON))
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)
	assert.Contains(t, response, "client_id")
}

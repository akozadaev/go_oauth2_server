package storage

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"go_oauth2_server/internal/models"

	"github.com/go-oauth2/oauth2/v4"
	oauthModels "github.com/go-oauth2/oauth2/v4/models"
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

func TestPostgresStore_CreateClient(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	// Очищаем данные перед тестом
	cleanupTestDB(db)
	defer cleanupTestDB(db)

	store := NewPostgresStore(db)

	ctx := context.Background()
	client := &models.Client{
		ID:        "test-client",
		Secret:    "test-secret",
		Domain:    "http://localhost:3000",
		UserID:    "test-user",
		CreatedAt: time.Now(),
	}

	err := store.CreateClient(ctx, client)
	assert.NoError(t, err)

	// Проверяем что клиент создан
	retrievedClient, err := store.GetClient(ctx, client.ID)
	assert.NoError(t, err)
	assert.Equal(t, client.ID, retrievedClient.ID)
	assert.Equal(t, client.Secret, retrievedClient.Secret)
	assert.Equal(t, client.Domain, retrievedClient.Domain)
	assert.Equal(t, client.UserID, retrievedClient.UserID)
}

func TestPostgresStore_CreateUser(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	// Очищаем данные перед тестом
	cleanupTestDB(db)
	defer cleanupTestDB(db)

	store := NewPostgresStore(db)

	ctx := context.Background()
	user := &models.User{
		ID:        "test-user-create",
		Username:  "testuser-create",
		Password:  "testpassword",
		CreatedAt: time.Now(),
	}

	err := store.CreateUser(ctx, user)
	assert.NoError(t, err)

	// Проверяем что пользователь создан
	retrievedUser, err := store.GetUser(ctx, user.Username)
	assert.NoError(t, err)
	assert.NotNil(t, retrievedUser)
	assert.Equal(t, user.Username, retrievedUser.Username)
}

func TestPostgresStore_ValidateUser(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	// Очищаем данные перед тестом
	cleanupTestDB(db)
	defer cleanupTestDB(db)

	store := NewPostgresStore(db)

	ctx := context.Background()
	user := &models.User{
		ID:        "test-user-validate",
		Username:  "testuser-validate",
		Password:  "testpassword",
		CreatedAt: time.Now(),
	}

	err := store.CreateUser(ctx, user)
	assert.NoError(t, err)

	// Проверяем валидацию пользователя
	validatedUser, err := store.ValidateUser(ctx, user.Username, user.Password)
	assert.NoError(t, err)
	assert.NotNil(t, validatedUser)
	assert.Equal(t, user.Username, validatedUser.Username)

	// Проверяем неверный пароль
	_, err = store.ValidateUser(ctx, user.Username, "wrongpassword")
	assert.Error(t, err)
}

func TestPostgresStore_ValidateClient(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	// Очищаем данные перед тестом
	cleanupTestDB(db)
	defer cleanupTestDB(db)

	store := NewPostgresStore(db)

	ctx := context.Background()
	client := &models.Client{
		ID:        "test-client-validate",
		Secret:    "test-secret",
		Domain:    "http://localhost:3000",
		UserID:    "test-user",
		CreatedAt: time.Now(),
	}

	err := store.CreateClient(ctx, client)
	assert.NoError(t, err)

	// Проверяем валидацию клиента
	validatedClient, err := store.ValidateClient(ctx, client.ID, client.Secret)
	assert.NoError(t, err)
	assert.NotNil(t, validatedClient)
	assert.Equal(t, client.ID, validatedClient.ID)
	assert.Equal(t, client.Secret, validatedClient.Secret)

	// Проверяем неверный секрет
	_, err = store.ValidateClient(ctx, client.ID, "wrong-secret")
	assert.Error(t, err)
}

func TestClientStore_GetByID(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	// Очищаем данные перед тестом
	cleanupTestDB(db)
	defer cleanupTestDB(db)

	clientStore := &ClientStore{
		db:      db,
		clients: make(map[string]oauth2.ClientInfo),
		logger:  slog.New(slog.NewJSONHandler(os.Stdout, nil)),
	}

	ctx := context.Background()
	clientInfo := &oauthModels.Client{
		ID:     "test-client",
		Secret: "test-secret",
		Domain: "http://localhost:3000",
		UserID: "test-user",
	}

	err := clientStore.Set(ctx, clientInfo.GetID(), clientInfo)
	assert.NoError(t, err)

	// Проверяем получение клиента
	retrievedClient, err := clientStore.GetByID(ctx, clientInfo.GetID())
	assert.NoError(t, err)
	assert.Equal(t, clientInfo.GetID(), retrievedClient.GetID())
	assert.Equal(t, clientInfo.GetSecret(), retrievedClient.GetSecret())
	assert.Equal(t, clientInfo.GetDomain(), retrievedClient.GetDomain())
}

func TestPostgresStore_Integration(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	// Очищаем данные перед тестом
	cleanupTestDB(db)
	defer cleanupTestDB(db)

	store := NewPostgresStore(db)

	ctx := context.Background()

	// Создаем пользователя
	user := &models.User{
		ID:        "test-user-integration",
		Username:  "testuser-integration",
		Password:  "testpassword",
		CreatedAt: time.Now(),
	}

	err := store.CreateUser(ctx, user)
	assert.NoError(t, err)

	// Создаем клиента
	client := &models.Client{
		ID:        "test-client-integration",
		Secret:    "test-secret",
		Domain:    "http://localhost:3000",
		UserID:    user.ID,
		CreatedAt: time.Now(),
	}

	err = store.CreateClient(ctx, client)
	assert.NoError(t, err)

	// Проверяем что все работает вместе
	validatedUser, err := store.ValidateUser(ctx, user.Username, user.Password)
	assert.NoError(t, err)
	assert.NotNil(t, validatedUser)
	assert.Equal(t, user.Username, validatedUser.Username)

	validatedClient, err := store.ValidateClient(ctx, client.ID, client.Secret)
	assert.NoError(t, err)
	assert.NotNil(t, validatedClient)
	assert.Equal(t, client.ID, validatedClient.ID)

	// Проверяем ping
	err = store.Ping(ctx)
	assert.NoError(t, err)
}

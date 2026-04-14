package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"log"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"go_oauth2_server/internal/config"
	"go_oauth2_server/internal/models"
	"go_oauth2_server/internal/storage"
	"go_oauth2_server/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var handlersTestDB *sql.DB

func TestMain(m *testing.M) {
	db, cleanup, err := testutil.StartPostgres()
	if err != nil {
		log.Printf("Docker unavailable, skipping tests: %v", err)
		os.Exit(0)
	}

	handlersTestDB = db

	code := m.Run()
	if err := db.Close(); err != nil {
		log.Printf("Failed to close DB: %v", err)
	}
	cleanup()
	os.Exit(code)
}

func setupTestHandler(t *testing.T) *Handler {
	require.NotNil(t, handlersTestDB, "database not initialized (Docker may be unavailable)")
	require.NoError(t, testutil.CleanupTestDB(handlersTestDB))

	store := storage.NewPostgresStore(handlersTestDB)
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg := &config.Config{
		Port:              "8080",
		JWTSecret:         "test-secret",
		TokenExpiration:   60 * time.Minute,
		RefreshExpiration: 168 * time.Hour,
		LogLevel:          "info",
	}

	return New(store, logger, cfg)
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
	assert.Contains(t, response, "client_id")
}

func TestHandler_Integration(t *testing.T) {
	handler := setupTestHandler(t)

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

	healthReq := httptest.NewRequest("GET", "/health", nil)
	healthW := httptest.NewRecorder()
	handler.Health(healthW, healthReq)
	assert.Equal(t, http.StatusOK, healthW.Code)
}

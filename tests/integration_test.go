package tests

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"log"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"go_oauth2_server/internal/config"
	"go_oauth2_server/internal/handlers"
	"go_oauth2_server/internal/models"
	"go_oauth2_server/internal/storage"
	"go_oauth2_server/testutil"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var integrationTestDB *sql.DB

func TestMain(m *testing.M) {
	db, cleanup, err := testutil.StartPostgres()
	if err != nil {
		log.Printf("Docker unavailable, skipping tests: %v", err)
		os.Exit(0)
	}
	defer func() {
		db.Close()
		cleanup()
	}()

	integrationTestDB = db

	code := m.Run()
	os.Exit(code)
}

func setupTestServer(t *testing.T) *httptest.Server {
	require.NotNil(t, integrationTestDB, "database not initialized (Docker may be unavailable)")
	require.NoError(t, testutil.CleanupTestDB(integrationTestDB))

	store := storage.NewPostgresStore(integrationTestDB)
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

	return httptest.NewServer(router)
}

func TestOAuth2Flow_Integration(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

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

	var clientRespData map[string]interface{}
	err = json.NewDecoder(clientResp.Body).Decode(&clientRespData)
	require.NoError(t, err)
	clientID := clientRespData["client_id"].(string)
	clientSecret := clientRespData["client_secret"].(string)

	// Password grant flow
	tokenForm := url.Values{}
	tokenForm.Set("grant_type", "password")
	tokenForm.Set("client_id", clientID)
	tokenForm.Set("client_secret", clientSecret)
	tokenForm.Set("username", userData.Username)
	tokenForm.Set("password", userData.Password)

	tokenResp, err := http.Post(server.URL+"/token", "application/x-www-form-urlencoded", strings.NewReader(tokenForm.Encode()))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, tokenResp.StatusCode)

	var tokenResponse map[string]interface{}
	err = json.NewDecoder(tokenResp.Body).Decode(&tokenResponse)
	require.NoError(t, err)
	assert.Contains(t, tokenResponse, "access_token")
	assert.Contains(t, tokenResponse, "token_type")
	assert.Contains(t, tokenResponse, "expires_in")

	accessToken, ok := tokenResponse["access_token"].(string)
	require.True(t, ok)
	assert.NotEmpty(t, accessToken)

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

package storage

import (
	"context"
	"database/sql"
	"log"
	"os"
	"testing"
	"time"

	"go_oauth2_server/internal/models"
	"go_oauth2_server/testutil"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var storageTestDB *sql.DB

func TestMain(m *testing.M) {
	db, cleanup, err := testutil.StartPostgres()
	if err != nil {
		log.Printf("Docker unavailable, skipping tests: %v", err)
		os.Exit(0)
	}

	storageTestDB = db

	code := m.Run()
	if err := db.Close(); err != nil {
		log.Printf("Failed to close DB: %v", err)
	}
	cleanup()
	os.Exit(code)
}

func getTestDB(t *testing.T) *sql.DB {
	require.NotNil(t, storageTestDB, "database not initialized (Docker may be unavailable)")
	require.NoError(t, testutil.CleanupTestDB(storageTestDB))
	return storageTestDB
}

func TestPostgresStore_CreateClient(t *testing.T) {
	db := getTestDB(t)

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

	retrievedClient, err := store.GetClient(ctx, client.ID)
	assert.NoError(t, err)
	assert.Equal(t, client.ID, retrievedClient.ID)
	assert.Equal(t, client.Secret, retrievedClient.Secret)
	assert.Equal(t, client.Domain, retrievedClient.Domain)
	assert.Equal(t, client.UserID, retrievedClient.UserID)
}

func TestPostgresStore_CreateUser(t *testing.T) {
	db := getTestDB(t)

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

	retrievedUser, err := store.GetUser(ctx, user.Username)
	assert.NoError(t, err)
	assert.NotNil(t, retrievedUser)
	assert.Equal(t, user.Username, retrievedUser.Username)
	assert.Equal(t, []string{models.RoleUser}, retrievedUser.Roles)
}

func TestPostgresStore_ValidateUser(t *testing.T) {
	db := getTestDB(t)

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

	validatedUser, err := store.ValidateUser(ctx, user.Username, user.Password)
	assert.NoError(t, err)
	assert.NotNil(t, validatedUser)
	assert.Equal(t, user.Username, validatedUser.Username)

	_, err = store.ValidateUser(ctx, user.Username, "wrongpassword")
	assert.Error(t, err)
}

func TestPostgresStore_ValidateClient(t *testing.T) {
	db := getTestDB(t)

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

	validatedClient, err := store.ValidateClient(ctx, client.ID, client.Secret)
	assert.NoError(t, err)
	assert.NotNil(t, validatedClient)
	assert.Equal(t, client.ID, validatedClient.ID)
	assert.Equal(t, client.Secret, validatedClient.Secret)

	_, err = store.ValidateClient(ctx, client.ID, "wrong-secret")
	assert.Error(t, err)
}

func TestClientStore_GetByID(t *testing.T) {
	db := getTestDB(t)

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

	clientStore := store.GetClientStore().(*ClientStore)
	require.NotNil(t, clientStore)

	retrievedClient, err := clientStore.GetByID(ctx, client.ID)
	assert.NoError(t, err)
	assert.Equal(t, client.ID, retrievedClient.GetID())
	assert.Equal(t, client.Secret, retrievedClient.GetSecret())
	assert.Equal(t, client.Domain, retrievedClient.GetDomain())
}

func TestClientStore_GetByID_NotFound(t *testing.T) {
	db := getTestDB(t)

	store := NewPostgresStore(db)
	clientStore := store.GetClientStore().(*ClientStore)

	ctx := context.Background()
	_, err := clientStore.GetByID(ctx, "nonexistent-client-id")
	assert.Error(t, err)
}

func TestPostgresStore_Integration(t *testing.T) {
	db := getTestDB(t)

	store := NewPostgresStore(db)

	ctx := context.Background()

	user := &models.User{
		ID:        "test-user-integration",
		Username:  "testuser-integration",
		Password:  "testpassword",
		CreatedAt: time.Now(),
	}

	err := store.CreateUser(ctx, user)
	assert.NoError(t, err)

	client := &models.Client{
		ID:        "test-client-integration",
		Secret:    "test-secret",
		Domain:    "http://localhost:3000",
		UserID:    user.ID,
		CreatedAt: time.Now(),
	}

	err = store.CreateClient(ctx, client)
	assert.NoError(t, err)

	validatedUser, err := store.ValidateUser(ctx, user.Username, user.Password)
	assert.NoError(t, err)
	assert.NotNil(t, validatedUser)
	assert.Equal(t, user.Username, validatedUser.Username)

	validatedClient, err := store.ValidateClient(ctx, client.ID, client.Secret)
	assert.NoError(t, err)
	assert.NotNil(t, validatedClient)
	assert.Equal(t, client.ID, validatedClient.ID)

	err = store.Ping(ctx)
	assert.NoError(t, err)
}

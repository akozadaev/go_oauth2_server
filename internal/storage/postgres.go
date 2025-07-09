package storage

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"go_oauth2_server/internal/models"

	"github.com/go-oauth2/oauth2/v4"
	oauthModels "github.com/go-oauth2/oauth2/v4/models"
	"golang.org/x/crypto/bcrypt"
)

type PostgresStore struct {
	db          *sql.DB
	clientStore oauth2.ClientStore
	tokenStore  oauth2.TokenStore
	logger      *slog.Logger
}

func NewPostgresStore(db *sql.DB) *PostgresStore {
	logger := slog.Default()
	clientStore := &ClientStore{db: db, logger: logger}
	var tokenStore oauth2.TokenStore
	if logger != nil { // TODO  Подумать о реализации. Пока так оставлю
		tokenStore = NewProductionTokenStore(db, logger) // Продакшн
	} else {
		tokenStore = NewSimpleTokenStore(db) // Разработка
	}

	return &PostgresStore{
		db:          db,
		clientStore: clientStore,
		tokenStore:  tokenStore,
		logger:      logger,
	}
}

func (s *PostgresStore) GetClientStore() oauth2.ClientStore {
	return s.clientStore
}

func (s *PostgresStore) GetTokenStore() oauth2.TokenStore {
	return s.tokenStore
}

func (s *PostgresStore) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

func (s *PostgresStore) CreateClient(ctx context.Context, client *models.Client) error {
	query := `
        INSERT INTO clients (id, secret, domain, user_id, created_at)
        VALUES ($1, $2, $3, $4, $5)
    `
	_, err := s.db.ExecContext(ctx, query, client.ID, client.Secret, client.Domain, client.UserID, client.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	// Also add to OAuth2 client store
	clientInfo := &oauthModels.Client{
		ID:     client.ID,
		Secret: client.Secret,
		Domain: client.Domain,
		UserID: client.UserID,
	}

	if cs, ok := s.clientStore.(*ClientStore); ok {
		return cs.Set(ctx, client.ID, clientInfo)
	}

	return nil
}

func (s *PostgresStore) GetClient(ctx context.Context, clientID string) (*models.Client, error) {
	client := &models.Client{}
	query := `
        SELECT id, secret, domain, user_id, created_at
        FROM clients
        WHERE id = $1
    `
	err := s.db.QueryRowContext(ctx, query, clientID).Scan(
		&client.ID, &client.Secret, &client.Domain, &client.UserID, &client.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get client: %w", err)
	}
	return client, nil
}

func (s *PostgresStore) CreateUser(ctx context.Context, user *models.User) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	query := `
        INSERT INTO users (id, username, password, created_at)
        VALUES ($1, $2, $3, $4)
    `
	_, err = s.db.ExecContext(ctx, query, user.ID, user.Username, string(hashedPassword), user.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}
	return nil
}

func (s *PostgresStore) GetUser(ctx context.Context, username string) (*models.User, error) {
	user := &models.User{}
	query := `
        SELECT id, username, password, created_at
        FROM users
        WHERE username = $1
    `
	err := s.db.QueryRowContext(ctx, query, username).Scan(
		&user.ID, &user.Username, &user.Password, &user.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return user, nil
}

func (s *PostgresStore) ValidateUser(ctx context.Context, username, password string) (*models.User, error) {
	user, err := s.GetUser(ctx, username)
	if err != nil {
		return nil, err
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	return user, nil
}

func (s *PostgresStore) ValidateClient(ctx context.Context, clientID, clientSecret string) (*models.Client, error) {
	client, err := s.GetClient(ctx, clientID)
	if err != nil {
		return nil, err
	}

	if client.Secret != clientSecret {
		return nil, fmt.Errorf("invalid client credentials")
	}

	return client, nil
}

// CleanExpiredTokens очищает истекшие токены
func (s *PostgresStore) CleanExpiredTokens(ctx context.Context) error {
	if tokenStore, ok := s.tokenStore.(*SimpleTokenStore); ok {
		return tokenStore.CleanExpiredTokens(ctx)
	}
	return nil
}

// GetTokenStats возвращает статистику токенов
func (s *PostgresStore) GetTokenStats(ctx context.Context) (map[string]int64, error) {
	query := `
        SELECT 
            COUNT(*) as total_tokens,
            COUNT(CASE WHEN access_expires_at > NOW() THEN 1 END) as active_tokens,
            COUNT(CASE WHEN access_expires_at <= NOW() THEN 1 END) as expired_tokens
        FROM oauth2_tokens
    `

	var total, active, expired int64
	err := s.db.QueryRowContext(ctx, query).Scan(&total, &active, &expired)
	if err != nil {
		return nil, fmt.Errorf("failed to get token stats: %w", err)
	}

	return map[string]int64{
		"total":   total,
		"active":  active,
		"expired": expired,
	}, nil
}

// ClientStore implements oauth2.ClientStore
type ClientStore struct {
	db      *sql.DB
	clients map[string]oauth2.ClientInfo
	logger  *slog.Logger
}

func (cs *ClientStore) GetByID(ctx context.Context, id string) (oauth2.ClientInfo, error) {
	// First check in-memory cache
	if cs.clients != nil {
		if client, exists := cs.clients[id]; exists {
			if cs.logger != nil {
				cs.logger.Debug("Client found in cache", "client_id", id)
			}
			return client, nil
		}
	}

	// Query database
	client := &oauthModels.Client{}
	query := `
        SELECT id, secret, domain, user_id
        FROM clients
        WHERE id = $1
    `
	err := cs.db.QueryRowContext(ctx, query, id).Scan(
		&client.ID, &client.Secret, &client.Domain, &client.UserID,
	)
	if err != nil {
		if cs.logger != nil {
			cs.logger.Error("Failed to get client by ID", "client_id", id, "error", err)
		}
		return nil, fmt.Errorf("failed to get client by ID: %w", err)
	}

	if cs.logger != nil {
		cs.logger.Debug("Client retrieved from database", "client_id", id)
	}

	return client, nil
}

func (cs *ClientStore) Set(ctx context.Context, id string, client oauth2.ClientInfo) error {
	if cs.clients == nil {
		cs.clients = make(map[string]oauth2.ClientInfo)
	}
	cs.clients[id] = client

	if cs.logger != nil {
		cs.logger.Debug("Client cached", "client_id", id)
	}

	return nil
}

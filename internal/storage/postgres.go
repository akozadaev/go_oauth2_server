// Package storage реализует хранилища OAuth2-клиентов, пользователей и токенов.
package storage

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"go_oauth2_server/internal/models"

	"github.com/go-oauth2/oauth2/v4"
	oauthModels "github.com/go-oauth2/oauth2/v4/models"
	"github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

// PostgresStore инкапсулирует доступ к PostgreSQL для пользователей, клиентов и токенов OAuth2.
type PostgresStore struct {
	db          *sql.DB
	clientStore oauth2.ClientStore
	tokenStore  oauth2.TokenStore
	logger      *slog.Logger
}

// NewPostgresStore создает хранилище данных на PostgreSQL.
func NewPostgresStore(db *sql.DB) *PostgresStore {
	logger := slog.Default()
	// ClientStore теперь stateless (без in-memory кеша) для поддержки горизонтального масштабирования
	clientStore := &ClientStore{db: db, logger: logger}
	var tokenStore oauth2.TokenStore
	if logger != nil {
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

// GetClientStore возвращает реализацию oauth2.ClientStore.
func (s *PostgresStore) GetClientStore() oauth2.ClientStore {
	return s.clientStore
}

// GetTokenStore возвращает реализацию oauth2.TokenStore.
func (s *PostgresStore) GetTokenStore() oauth2.TokenStore {
	return s.tokenStore
}

// Ping проверяет доступность подключения к базе данных.
func (s *PostgresStore) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

// CreateClient сохраняет клиента OAuth2 в базе данных.
func (s *PostgresStore) CreateClient(ctx context.Context, client *models.Client) error {
	query := `
        INSERT INTO clients (id, secret, domain, user_id, created_at)
        VALUES ($1, $2, $3, $4, $5)
    `
	_, err := s.db.ExecContext(ctx, query, client.ID, client.Secret, client.Domain, client.UserID, client.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	return nil
}

// GetClient загружает клиента OAuth2 по идентификатору.
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

// CreateUser создает пользователя с хешированным паролем.
func (s *PostgresStore) CreateUser(ctx context.Context, user *models.User) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	roles := user.Roles
	if len(roles) == 0 {
		roles = models.DefaultUserRoles()
	}

	query := `
        INSERT INTO users (id, username, password, created_at, roles)
        VALUES ($1, $2, $3, $4, $5)
    `
	_, err = s.db.ExecContext(ctx, query, user.ID, user.Username, string(hashedPassword), user.CreatedAt, pq.Array(roles))
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}
	user.Roles = roles
	return nil
}

// GetUser загружает пользователя по имени.
func (s *PostgresStore) GetUser(ctx context.Context, username string) (*models.User, error) {
	user := &models.User{}
	var roles pq.StringArray
	query := `
        SELECT id, username, password, created_at, roles
        FROM users
        WHERE username = $1
    `
	err := s.db.QueryRowContext(ctx, query, username).Scan(
		&user.ID, &user.Username, &user.Password, &user.CreatedAt, &roles,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	user.Roles = append([]string(nil), []string(roles)...)
	return user, nil
}

// GetUserRolesByID возвращает роли пользователя по id (для выдачи JWT).
func (s *PostgresStore) GetUserRolesByID(ctx context.Context, id string) ([]string, error) {
	var roles pq.StringArray
	err := s.db.QueryRowContext(ctx, `SELECT roles FROM users WHERE id = $1`, id).Scan(&roles)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found: %w", err)
		}
		return nil, fmt.Errorf("failed to get user roles: %w", err)
	}
	return append([]string(nil), []string(roles)...), nil
}

// ValidateUser проверяет учетные данные пользователя.
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

// ValidateClient проверяет учетные данные OAuth2-клиента.
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
// Примечание: In-memory кеш убран для поддержки горизонтального масштабирования.
// Вместо этого всегда обращаемся к БД, которая имеет индекс на id (PRIMARY KEY).
// Для дальнейшей оптимизации можно добавить Redis для распределенного кеширования.
type ClientStore struct {
	db     *sql.DB
	logger *slog.Logger
}

// GetByID возвращает клиента по id из базы данных.
func (cs *ClientStore) GetByID(ctx context.Context, id string) (oauth2.ClientInfo, error) {
	// Всегда обращаемся к БД для поддержки горизонтального масштабирования
	// PRIMARY KEY на id обеспечивает быстрый поиск
	client := &oauthModels.Client{}
	query := `
        SELECT id, secret, domain, user_id
        FROM clients
        WHERE id = $1
    `

	// Добавляем таймаут для запроса
	queryCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	err := cs.db.QueryRowContext(queryCtx, query, id).Scan(
		&client.ID, &client.Secret, &client.Domain, &client.UserID,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			if cs.logger != nil {
				cs.logger.Debug("Client not found", "client_id", id)
			}
			return nil, fmt.Errorf("client not found: %w", err)
		}
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

// Set реализует интерфейс oauth2.ClientStore и является no-op для stateless-хранилища.
func (cs *ClientStore) Set(_ context.Context, id string, _ oauth2.ClientInfo) error {
	// Метод Set больше не используется для кеширования,
	// так как мы всегда обращаемся к БД напрямую
	// Оставлен для совместимости с интерфейсом oauth2.ClientStore
	if cs.logger != nil {
		cs.logger.Debug("ClientStore.Set called (no-op for stateless store)", "client_id", id)
	}
	return nil
}

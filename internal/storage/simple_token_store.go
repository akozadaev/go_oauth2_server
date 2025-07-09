package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/go-oauth2/oauth2/v4"
	"github.com/go-oauth2/oauth2/v4/models"
)

// SimpleTokenStore - упрощенная версия token store для PostgreSQL
type SimpleTokenStore struct {
	db *sql.DB
}

// NewSimpleTokenStore создает новый простой PostgreSQL token store
func NewSimpleTokenStore(db *sql.DB) *SimpleTokenStore {
	return &SimpleTokenStore{db: db}
}

// Create создает новый токен
func (ts *SimpleTokenStore) Create(ctx context.Context, info oauth2.TokenInfo) error {
	query := `
        INSERT INTO oauth2_tokens (
            access_token, refresh_token, client_id, user_id, scope,
            access_expires_at, refresh_expires_at
        ) VALUES ($1, $2, $3, $4, $5, $6, $7)
        ON CONFLICT (access_token) DO UPDATE SET
            refresh_token = EXCLUDED.refresh_token,
            scope = EXCLUDED.scope,
            access_expires_at = EXCLUDED.access_expires_at,
            refresh_expires_at = EXCLUDED.refresh_expires_at,
            updated_at = NOW()
    `

	// Вычисляем время истечения токенов
	accessExpiresAt := info.GetAccessCreateAt().Add(info.GetAccessExpiresIn())

	var refreshExpiresAt *time.Time
	if info.GetRefresh() != "" && info.GetRefreshExpiresIn() > 0 {
		expires := info.GetRefreshCreateAt().Add(info.GetRefreshExpiresIn())
		refreshExpiresAt = &expires
	}

	_, err := ts.db.ExecContext(ctx, query,
		info.GetAccess(),
		info.GetRefresh(),
		info.GetClientID(),
		info.GetUserID(),
		info.GetScope(),
		accessExpiresAt,
		refreshExpiresAt,
	)

	return err
}

// RemoveByAccess удаляет токен по access token
func (ts *SimpleTokenStore) RemoveByAccess(ctx context.Context, access string) error {
	query := `DELETE FROM oauth2_tokens WHERE access_token = $1`
	_, err := ts.db.ExecContext(ctx, query, access)
	return err
}

// RemoveByRefresh удаляет токен по refresh token
func (ts *SimpleTokenStore) RemoveByRefresh(ctx context.Context, refresh string) error {
	query := `DELETE FROM oauth2_tokens WHERE refresh_token = $1`
	_, err := ts.db.ExecContext(ctx, query, refresh)
	return err
}

// RemoveByCode удаляет токен по authorization code
func (ts *SimpleTokenStore) RemoveByCode(ctx context.Context, code string) error {
	// В нашей реализации authorization codes хранятся отдельно
	return nil
}

// GetByAccess получает токен по access token
func (ts *SimpleTokenStore) GetByAccess(ctx context.Context, access string) (oauth2.TokenInfo, error) {
	query := `
        SELECT access_token, refresh_token, client_id, user_id, scope,
               access_expires_at, refresh_expires_at, created_at
        FROM oauth2_tokens 
        WHERE access_token = $1 AND access_expires_at > NOW()
    `

	var accessToken, refreshToken, clientID, userID, scope string
	var accessExpiresAt, createdAt time.Time
	var refreshExpiresAt sql.NullTime

	err := ts.db.QueryRowContext(ctx, query, access).Scan(
		&accessToken,
		&refreshToken,
		&clientID,
		&userID,
		&scope,
		&accessExpiresAt,
		&refreshExpiresAt,
		&createdAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Токен не найден или истек
		}
		return nil, fmt.Errorf("failed to get token by access: %w", err)
	}

	// Создаем токен
	token := &models.Token{
		ClientID:        clientID,
		UserID:          userID,
		Access:          accessToken,
		Refresh:         refreshToken,
		Scope:           scope,
		AccessCreateAt:  createdAt,
		AccessExpiresIn: accessExpiresAt.Sub(createdAt),
	}

	if refreshExpiresAt.Valid {
		token.RefreshCreateAt = createdAt
		token.RefreshExpiresIn = refreshExpiresAt.Time.Sub(createdAt)
	}

	return token, nil
}

// GetByRefresh получает токен по refresh token
func (ts *SimpleTokenStore) GetByRefresh(ctx context.Context, refresh string) (oauth2.TokenInfo, error) {
	query := `
        SELECT access_token, refresh_token, client_id, user_id, scope,
               access_expires_at, refresh_expires_at, created_at
        FROM oauth2_tokens 
        WHERE refresh_token = $1 AND (refresh_expires_at IS NULL OR refresh_expires_at > NOW())
    `

	var accessToken, refreshToken, clientID, userID, scope string
	var accessExpiresAt, createdAt time.Time
	var refreshExpiresAt sql.NullTime

	err := ts.db.QueryRowContext(ctx, query, refresh).Scan(
		&accessToken,
		&refreshToken,
		&clientID,
		&userID,
		&scope,
		&accessExpiresAt,
		&refreshExpiresAt,
		&createdAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Токен не найден или истек
		}
		return nil, fmt.Errorf("failed to get token by refresh: %w", err)
	}

	// Создаем токен
	token := &models.Token{
		ClientID:        clientID,
		UserID:          userID,
		Access:          accessToken,
		Refresh:         refreshToken,
		Scope:           scope,
		AccessCreateAt:  createdAt,
		AccessExpiresIn: accessExpiresAt.Sub(createdAt),
	}

	if refreshExpiresAt.Valid {
		token.RefreshCreateAt = createdAt
		token.RefreshExpiresIn = refreshExpiresAt.Time.Sub(createdAt)
	}

	return token, nil
}

// GetByCode получает токен по authorization code
func (ts *SimpleTokenStore) GetByCode(ctx context.Context, code string) (oauth2.TokenInfo, error) {
	return nil, nil
}

// CleanExpiredTokens очищает истекшие токены
func (ts *SimpleTokenStore) CleanExpiredTokens(ctx context.Context) error {
	query := `
        DELETE FROM oauth2_tokens 
        WHERE access_expires_at < NOW() 
        AND (refresh_expires_at IS NULL OR refresh_expires_at < NOW())
    `

	result, err := ts.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to clean expired tokens: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected > 0 {
		fmt.Printf("Cleaned %d expired tokens\n", rowsAffected)
	}

	return nil
}

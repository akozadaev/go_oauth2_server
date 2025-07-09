package storage

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/go-oauth2/oauth2/v4"
	"github.com/go-oauth2/oauth2/v4/models"
)

// ProductionTokenStore - продакшн версия token store для PostgreSQL
type ProductionTokenStore struct {
	db     *sql.DB
	logger *slog.Logger
}

// NewProductionTokenStore создает новый продакшн PostgreSQL token store
func NewProductionTokenStore(db *sql.DB, logger *slog.Logger) *ProductionTokenStore {
	return &ProductionTokenStore{
		db:     db,
		logger: logger,
	}
}

// Create создает новый токен с детальным логированием
func (ts *ProductionTokenStore) Create(ctx context.Context, info oauth2.TokenInfo) error {
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

	// Время истечения токенов
	accessExpiresAt := info.GetAccessCreateAt().Add(info.GetAccessExpiresIn())

	var refreshExpiresAt *time.Time
	if info.GetRefresh() != "" && info.GetRefreshExpiresIn() > 0 {
		expires := info.GetRefreshCreateAt().Add(info.GetRefreshExpiresIn())
		refreshExpiresAt = &expires
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := ts.db.ExecContext(ctx, query,
		info.GetAccess(),
		info.GetRefresh(),
		info.GetClientID(),
		info.GetUserID(),
		info.GetScope(),
		accessExpiresAt,
		refreshExpiresAt,
	)

	if err != nil {
		ts.logger.Error("Failed to create token",
			"client_id", info.GetClientID(),
			"user_id", info.GetUserID(),
			"error", err,
		)
		return fmt.Errorf("failed to create token: %w", err)
	}

	ts.logger.Info("Token created successfully",
		"client_id", info.GetClientID(),
		"user_id", info.GetUserID(),
		"access_expires_at", accessExpiresAt,
	)

	return nil
}

// GetByAccess получает токен по access token с кешированием
func (ts *ProductionTokenStore) GetByAccess(ctx context.Context, access string) (oauth2.TokenInfo, error) {
	query := `
        SELECT access_token, refresh_token, client_id, user_id, scope,
               access_expires_at, refresh_expires_at, created_at
        FROM oauth2_tokens 
        WHERE access_token = $1 AND access_expires_at > NOW()
    `

	// Добавляем таймаут для запроса
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

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
			ts.logger.Debug("Token not found or expired", "access_token_prefix", access[:min(8, len(access))])
			return nil, nil
		}
		ts.logger.Error("Failed to get token by access", "error", err)
		return nil, fmt.Errorf("failed to get token by access: %w", err)
	}

	// Создаем токен с правильными типами
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

	ts.logger.Debug("Token retrieved successfully",
		"client_id", clientID,
		"user_id", userID,
	)

	return token, nil
}

// GetByRefresh получает токен по refresh token
func (ts *ProductionTokenStore) GetByRefresh(ctx context.Context, refresh string) (oauth2.TokenInfo, error) {
	query := `
        SELECT access_token, refresh_token, client_id, user_id, scope,
               access_expires_at, refresh_expires_at, created_at
        FROM oauth2_tokens 
        WHERE refresh_token = $1 AND (refresh_expires_at IS NULL OR refresh_expires_at > NOW())
    `

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

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
			ts.logger.Debug("Refresh token not found or expired")
			return nil, nil
		}
		ts.logger.Error("Failed to get token by refresh", "error", err)
		return nil, fmt.Errorf("failed to get token by refresh: %w", err)
	}

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

// RemoveByAccess удаляет токен по access token
func (ts *ProductionTokenStore) RemoveByAccess(ctx context.Context, access string) error {
	query := `DELETE FROM oauth2_tokens WHERE access_token = $1`

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	result, err := ts.db.ExecContext(ctx, query, access)
	if err != nil {
		ts.logger.Error("Failed to remove token by access", "error", err)
		return fmt.Errorf("failed to remove token by access: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected > 0 {
		ts.logger.Info("Token removed by access", "rows_affected", rowsAffected)
	}

	return nil
}

// RemoveByRefresh удаляет токен по refresh token
func (ts *ProductionTokenStore) RemoveByRefresh(ctx context.Context, refresh string) error {
	query := `DELETE FROM oauth2_tokens WHERE refresh_token = $1`

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	result, err := ts.db.ExecContext(ctx, query, refresh)
	if err != nil {
		ts.logger.Error("Failed to remove token by refresh", "error", err)
		return fmt.Errorf("failed to remove token by refresh: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected > 0 {
		ts.logger.Info("Token removed by refresh", "rows_affected", rowsAffected)
	}

	return nil
}

// RemoveByCode удаляет токен по authorization code
func (ts *ProductionTokenStore) RemoveByCode(ctx context.Context, code string) error {
	// В нашей реализации authorization codes хранятся отдельно
	return nil
}

// GetByCode получает токен по authorization code
func (ts *ProductionTokenStore) GetByCode(ctx context.Context, code string) (oauth2.TokenInfo, error) {
	return nil, nil
}

// CleanExpiredTokens очищает истекшие токены с детальной статистикой
func (ts *ProductionTokenStore) CleanExpiredTokens(ctx context.Context) error {
	query := `
        DELETE FROM oauth2_tokens 
        WHERE access_expires_at < NOW() 
        AND (refresh_expires_at IS NULL OR refresh_expires_at < NOW())
    `

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	start := time.Now()
	result, err := ts.db.ExecContext(ctx, query)
	if err != nil {
		ts.logger.Error("Failed to clean expired tokens", "error", err)
		return fmt.Errorf("failed to clean expired tokens: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	duration := time.Since(start)

	ts.logger.Info("Expired tokens cleaned",
		"rows_affected", rowsAffected,
		"duration", duration,
	)

	return nil
}

// GetTokenStats возвращает статистику токенов
func (ts *ProductionTokenStore) GetTokenStats(ctx context.Context) (map[string]int64, error) {
	query := `
        SELECT 
            COUNT(*) as total_tokens,
            COUNT(CASE WHEN access_expires_at > NOW() THEN 1 END) as active_tokens,
            COUNT(CASE WHEN access_expires_at <= NOW() THEN 1 END) as expired_tokens,
            COUNT(CASE WHEN refresh_token IS NOT NULL THEN 1 END) as with_refresh
        FROM oauth2_tokens
    `

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var total, active, expired, withRefresh int64
	err := ts.db.QueryRowContext(ctx, query).Scan(&total, &active, &expired, &withRefresh)
	if err != nil {
		return nil, fmt.Errorf("failed to get token stats: %w", err)
	}

	return map[string]int64{
		"total":        total,
		"active":       active,
		"expired":      expired,
		"with_refresh": withRefresh,
	}, nil
}

// Helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

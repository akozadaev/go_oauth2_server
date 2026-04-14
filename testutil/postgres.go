// Package testutil содержит вспомогательные утилиты для интеграционных тестов.
package testutil

import (
	"database/sql"
	"fmt"
	"log"
	"strings"

	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	// PostgreSQL driver for database/sql.
	_ "github.com/lib/pq"
)

const (
	postgresUser     = "test_user"
	postgresPassword = "test_password"
	postgresDB       = "test_db"
)

// StartPostgres запускает PostgreSQL в Docker-контейнере через dockertest.
// Возвращает *sql.DB, функцию очистки и ошибку.
// Функция очистки должна быть вызвана (например, через defer) после завершения тестов.
func StartPostgres() (*sql.DB, func(), error) {
	log.Println("testutil: connecting to Docker daemon…")
	pool, err := dockertest.NewPool("")
	if err != nil {
		return nil, nil, fmt.Errorf("could not construct pool: %w", err)
	}

	err = pool.Client.Ping()
	if err != nil {
		return nil, nil, fmt.Errorf("could not connect to Docker: %w", err)
	}

	log.Println("testutil: starting postgres:15-alpine (first run may take several minutes while the image is pulled)…")
	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "postgres",
		Tag:        "15-alpine",
		Env: []string{
			"POSTGRES_USER=" + postgresUser,
			"POSTGRES_PASSWORD=" + postgresPassword,
			"POSTGRES_DB=" + postgresDB,
		},
	}, func(config *docker.HostConfig) {
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{Name: "no"}
	})
	if err != nil {
		return nil, nil, fmt.Errorf("could not start resource: %w", err)
	}

	cleanup := func() {
		_ = pool.Purge(resource)
	}

	dbURL := fmt.Sprintf("postgres://%s:%s@localhost:%s/%s?sslmode=disable",
		postgresUser, postgresPassword, resource.GetPort("5432/tcp"), postgresDB)

	log.Println("testutil: waiting until PostgreSQL accepts connections…")
	var db *sql.DB
	if err := pool.Retry(func() error {
		var err error
		db, err = sql.Open("postgres", dbURL)
		if err != nil {
			return err
		}
		return db.Ping()
	}); err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("could not connect to database: %w", err)
	}

	if err := CreateTestTables(db); err != nil {
		_ = db.Close()
		cleanup()
		return nil, nil, fmt.Errorf("could not create test tables: %w", err)
	}

	log.Println("testutil: PostgreSQL is ready for tests")
	return db, cleanup, nil
}

// CreateTestTables создаёт тестовые таблицы в БД.
func CreateTestTables(db *sql.DB) error {
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
			roles TEXT[] NOT NULL DEFAULT ARRAY['ROLE_USER']::TEXT[],
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
			if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
				continue
			}
			return fmt.Errorf("failed to create table: %w", err)
		}
	}

	return nil
}

// CleanupTestDB очищает тестовые данные из БД.
func CleanupTestDB(db *sql.DB) error {
	queries := []string{
		"DELETE FROM oauth2_tokens",
		"DELETE FROM tokens",
		"DELETE FROM clients",
		"DELETE FROM users",
	}

	for _, query := range queries {
		if _, err := db.Exec(query); err != nil {
			if !strings.Contains(err.Error(), "does not exist") {
				return fmt.Errorf("failed to cleanup table: %w", err)
			}
		}
	}

	return nil
}

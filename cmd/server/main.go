package main

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go_oauth2_server/internal/config"
	"go_oauth2_server/internal/handlers"
	"go_oauth2_server/internal/storage"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	// Настройка структурированного логирования с использованием slog (Go 1.21+)
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Загрузка переменных окружения
	if err := godotenv.Load(); err != nil {
		logger.Info("No .env file found, using environment variables")
	}

	// Инициализация конфигурации
	cfg := config.Load()

	// Подключение к базе данных с использованием контекста
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		logger.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}

	// Проверка подключения к базе данных
	if err := db.PingContext(ctx); err != nil {
		logger.Error("Failed to ping database", "error", err)
		os.Exit(1)
	}

	// Запуск миграций
	if err := runMigrations(cfg.DatabaseURL); err != nil {
		logger.Error("Failed to run migrations", "error", err)
		os.Exit(1)
	}

	// Инициализация хранилища
	store := storage.NewPostgresStore(db)

	// Инициализация обработчиков
	h := handlers.New(store, logger, cfg)

	// Настройка маршрутов
	router := mux.NewRouter()
	router.HandleFunc("/authorize", h.Authorize).Methods("GET", "POST")
	router.HandleFunc("/token", h.Token).Methods("POST")
	router.HandleFunc("/introspect", h.Introspect).Methods("POST")
	router.HandleFunc("/clients", h.RegisterClient).Methods("POST")
	router.HandleFunc("/health", h.Health).Methods("GET")
	router.HandleFunc("/users", h.RegisterUser).Methods("POST")

	// Добавление middleware
	router.Use(loggingMiddleware(logger))
	router.Use(corsMiddleware)

	// Настройка и запуск сервера
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Запуск сервера в отдельной горутине
	go func() {
		logger.Info("Server starting", "port", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("Server failed to start", "error", err)
			os.Exit(1)
		}
	}()

	// Ожидание сигнала завершения для корректного завершения работы
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("Shutting down server...")

	// Корректное завершение работы сервера с использованием контекста
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)

	if err = srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("Server forced to shutdown", "error", err)
		os.Exit(1)
	}

	defer func(db *sql.DB) {
		cancel()
		shutdownCancel()
		_ = db.Close()
	}(db)

	logger.Info("Server exited gracefully")
}

func runMigrations(databaseURL string) error {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return err
	}
	defer func(db *sql.DB) {
		_ = db.Close()
	}(db)

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return err
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://migrations",
		"postgres", driver)
	if err != nil {
		return err
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}

	return nil
}

func loggingMiddleware(logger *slog.Logger) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// ResponseWriter (для захвата кода состояния)
			wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			next.ServeHTTP(wrapped, r)

			logger.Info("Request processed",
				"method", r.Method,
				"path", r.URL.Path,
				"status", wrapped.statusCode,
				"duration", time.Since(start),
				"ip", r.RemoteAddr,
				"user_agent", r.UserAgent(),
			)
		})
	}
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// responseWriter — обёртка над http.ResponseWriter для захвата кода состояния
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

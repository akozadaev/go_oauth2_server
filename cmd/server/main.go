//go:build !debug

package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
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
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status"},
	)

	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint"},
	)

	oauth2TokensIssued = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "oauth2_tokens_issued_total",
			Help: "Total number of OAuth2 tokens issued",
		},
		[]string{"grant_type"},
	)

	oauth2TokensValidated = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "oauth2_tokens_validated_total",
			Help: "Total number of OAuth2 tokens validated",
		},
		[]string{"valid"},
	)
)

func init() {
	// Регистрируем метрики
	prometheus.MustRegister(httpRequestsTotal)
	prometheus.MustRegister(httpRequestDuration)
	prometheus.MustRegister(oauth2TokensIssued)
	prometheus.MustRegister(oauth2TokensValidated)
}

func main() {
	err := run()
	if err != nil {
		os.Exit(1)
	}
}

func run() error {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	if err := godotenv.Load(); err != nil {
		logger.Info("No .env file found, using environment variables")
	}

	cfg := config.Load()

	// Ожидание готовности БД до открытия соединения ()
	if err := waitForDB(cfg.DatabaseURL); err != nil {
		logger.Error("Database not ready", "error", err)
		return err
	}

	// Подключение к базе данных с таймаутом
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		logger.Error("Failed to connect to database", "error", err)
		return err
	}
	defer db.Close()

	if err := db.PingContext(ctx); err != nil {
		logger.Error("Failed to ping database", "error", err)
		return err
	}

	if err := runMigrations(cfg.DatabaseURL); err != nil {
		logger.Error("Failed to run migrations", "error", err)
		return err
	}

	store := storage.NewPostgresStore(db)
	h := handlers.New(store, logger, cfg)

	router := mux.NewRouter()
	router.HandleFunc("/authorize", h.Authorize).Methods("GET", "POST")
	router.HandleFunc("/token", h.Token).Methods("POST")
	router.HandleFunc("/introspect", h.Introspect).Methods("POST")
	router.HandleFunc("/clients", h.RegisterClient).Methods("POST")
	router.HandleFunc("/health", h.Health).Methods("GET")
	router.HandleFunc("/users", h.RegisterUser).Methods("POST")

	// Prometheus метрики
	router.Handle("/metrics", promhttp.Handler()).Methods("GET")

	router.Use(loggingMiddleware(logger))
	router.Use(corsMiddleware)
	router.Use(metricsMiddleware)

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Info("Server starting", "port", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("Server failed to start", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("Shutting down server...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("Server forced to shutdown", "error", err)
		return err
	}

	logger.Info("Server exited gracefully")
	return nil
}

func waitForDB(databaseURL string) error {
	const maxRetries = 10
	for i := 1; i <= maxRetries; i++ {
		db, err := sql.Open("postgres", databaseURL)
		if err == nil {
			if err = db.Ping(); err == nil {
				_ = db.Close()
				return nil
			}
		}
		if db != nil {
			_ = db.Close()
		}
		time.Sleep(3 * time.Second)
	}
	return fmt.Errorf("database not ready after %d attempts", maxRetries)
}

func runMigrations(databaseURL string) error {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return err
	}
	defer db.Close()

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return err
	}

	m, err := migrate.NewWithDatabaseInstance("file://migrations", "postgres", driver)
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

func metricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapped, r)

		duration := time.Since(start).Seconds()

		// Метрики запросов
		httpRequestsTotal.WithLabelValues(r.Method, r.URL.Path, fmt.Sprintf("%d", wrapped.statusCode)).Inc()
		httpRequestDuration.WithLabelValues(r.Method, r.URL.Path).Observe(duration)
	})
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

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

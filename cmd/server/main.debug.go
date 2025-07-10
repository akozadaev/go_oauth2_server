//go:build debug

// main.debug.go
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	fmt.Println("🚀 OAuth2 Server Debug Version Starting...")

	// Проверяем переменные окружения
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	dbURL := os.Getenv("DATABASE_URL")
	fmt.Printf("📊 Port: %s\n", port)
	fmt.Printf("📊 Database URL: %s\n", dbURL)

	// Простой HTTP сервер для тестирования
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"status":"healthy","timestamp":%d}`, time.Now().Unix())
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "OAuth2 Server Debug - OK")
	})

	fmt.Printf("🌐 Server starting on port %s...\n", port)

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("❌ Server failed to start: %v", err)
	}
}

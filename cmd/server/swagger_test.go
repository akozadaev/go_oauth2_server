//go:build !debug

package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestSwagger(t *testing.T) {
	router := chi.NewRouter()
	registerSwagger(router)
	for _, path := range []string{"/swagger/index.html", "/swagger/doc.json", "/swagger/swagger-ui.css", "/swagger/swagger-ui-bundle.js"} {
		t.Run(path, func(t *testing.T) {
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
			if response.Code != http.StatusOK {
				t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
			}
			if path == "/swagger/doc.json" {
				var spec struct {
					Swagger string                     `json:"swagger"`
					Paths   map[string]json.RawMessage `json:"paths"`
				}
				if err := json.Unmarshal(response.Body.Bytes(), &spec); err != nil {
					t.Fatal(err)
				}
				if spec.Swagger != "2.0" || len(spec.Paths) == 0 {
					t.Fatal("empty or invalid API specification")
				}
			}
			if path == "/swagger/index.html" && !strings.Contains(response.Body.String(), "doc.json") {
				t.Fatal("UI does not reference the local specification")
			}
		})
	}
}

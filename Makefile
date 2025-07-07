# 🛠 Установка всех утилит
tools:
	go install github.com/mgechev/revive@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.61.0

# 📦 Генерация всего, что помечено //go:generate
generate:
	go generate ./...

# ⚙️ Сборка сервера
build:
	CGO_ENABLED=0 go build -a -o go_oauth2_server ./cmd/server/

# 📦 Сборка для продакшена (Linux AMD64)
release:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "-s -w" -o go_oauth2_server ./cmd/server
	zip -9 -r ./go_oauth2_server.zip ./go_oauth2_server

# 🧹 Форматирование gofmt (автоисправление)
fmt:
	gofmt -s -w .

# 🧪 Тестирование
test:
	go test -v ./...

# 🧪 Покрытие тестами
test-coverage:
	go test -cover -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

# 🧼 Полный линтинг с golangci-lint (версия 2)
lint-full:
	@if ! [ -x "$$(command -v golangci-lint)" ]; then \
		echo "Installing golangci-lint..."; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin v1.61.0; \
	fi
	golangci-lint run ./...

# 🧼 Автофиксы
lint-fix:
	golangci-lint run --fix ./...

# 🧪 Финальная проверка перед коммитом
check: fmt lint-full test

# Makefile для OAuth2 сервера
.PHONY: help tools generate build release fmt test test-coverage lint-full lint-fix check clean-all clean-deps clean-deps-safe fix-network vendor stop-conflicts docker-build docker-build-debug docker-build-simple docker-build-offline up up-debug up-simple up-no-build down logs logs-server logs-db logs-redis status restart restart-server check-ports shell db-shell redis-shell docker-test diagnose diagnose-container health quick-start debug

# ==================== РАЗРАБОТКА ====================

tools: ## 🛠 Установка всех утилит
	go install github.com/mgechev/revive@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.61.0

generate: ## 📦 Генерация всего, что помечено //go:generate
	go generate ./...

build: ## ⚙️ Сборка сервера (локально)
	CGO_ENABLED=0 go build -a -o go_oauth2_server ./cmd/server/

build-debug: ## ⚙️ Сборка debug версии (локально)
	CGO_ENABLED=0 go build -a -o go_oauth2_server_debug ./cmd/server/main.debug.go

release: ## 📦 Сборка для продакшена (Linux AMD64)
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "-s -w" -o go_oauth2_server ./cmd/server/
	zip -9 -r ./go_oauth2_server.zip ./go_oauth2_server

fmt: ## 🧹 Форматирование gofmt (автоисправление)
	gofmt -s -w .

test: ## 🧪 Тестирование (локально)
	go test -v ./...

test-coverage: ## 🧪 Покрытие тестами
	go test -cover -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

lint-full: ## 🧼 Полный линтинг с golangci-lint
	@if ! [ -x "$$(command -v golangci-lint)" ]; then \
		echo "Installing golangci-lint..."; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin v1.61.0; \
	fi
	golangci-lint run ./...

lint-fix: ## 🧼 Автофиксы линтера
	golangci-lint run --fix ./...

check: fmt lint-full test ## 🧪 Финальная проверка перед коммитом

fix-network: ## 🌐 Исправление проблем с сетью
	@echo "🌐 Исправление проблем с сетью..."
	@chmod +x scripts/fix-network.sh
	@./scripts/fix-network.sh

clean-deps-safe: ## 🧹 Безопасная очистка зависимостей Go
	@echo "🧹 Безопасная очистка зависимостей Go..."
	@chmod +x scripts/clean-deps-safe.sh
	@./scripts/clean-deps-safe.sh

clean-deps: clean-deps-safe ## 🧹 Очистка зависимостей Go (алиас для безопасной версии)

vendor: ## 📦 Создание vendor директории
	@echo "📦 Создание vendor директории..."
	@go mod vendor
	@echo "✅ Vendor директория создана"

# ==================== DOCKER ====================

clean-all: ## 🧹 Полная очистка Docker
	@echo "🧹 Полная очистка Docker..."
	-docker-compose down -v --remove-orphans 2>/dev/null || true
	-docker-compose -f docker-compose.debug.yml down -v --remove-orphans 2>/dev/null || true
	-docker-compose -f docker-compose.simple.yml down -v --remove-orphans 2>/dev/null || true
	-docker container prune -f
	-docker volume prune -f
	-docker network prune -f
	@echo "✅ Очистка завершена"

stop-conflicts: ## 🛑 Остановить конфликтующие процессы
	@echo "🔍 Остановка конфликтующих процессов..."
	-sudo lsof -ti :5433 | xargs sudo kill -9 2>/dev/null || true
	-sudo lsof -ti :6380 | xargs sudo kill -9 2>/dev/null || true
	-sudo lsof -ti :8080 | xargs sudo kill -9 2>/dev/null || true
	-docker stop $$(docker ps -aq --filter "name=oauth2") 2>/dev/null || true
	-docker rm $$(docker ps -aq --filter "name=oauth2") 2>/dev/null || true
	@echo "✅ Конфликты устранены"

docker-build: clean-all clean-deps ## 🔨 Собрать Docker образы заново
	@echo "🔨 Сборка Docker образов..."
	docker-compose build --no-cache --force-rm
	@echo "✅ Docker образы собраны"

docker-build-debug: clean-all clean-deps ## 🔨 Собрать Docker образы для отладки
	@echo "🔨 Сборка Docker образов для отладки..."
	docker-compose -f docker-compose.debug.yml build --no-cache --force-rm
	@echo "✅ Docker образы для отладки собраны"

docker-build-simple: clean-all ## 🔨 Собрать простые Docker образы
	@echo "🔨 Сборка простых Docker образов..."
	docker-compose -f docker-compose.simple.yml build --no-cache --force-rm
	@echo "✅ Простые Docker образы собраны"

up: stop-conflicts docker-build ## 🚀 Запустить все сервисы
	@echo "🚀 Запуск сервисов..."
	docker-compose up -d
	@echo "⏳ Ожидание готовности сервисов (60 секунд)..."
	@sleep 60
	@make status
	@echo "✅ Сервисы запущены"

up-debug: stop-conflicts docker-build-debug ## 🚀 Запустить в режиме отладки
	@echo "🚀 Запуск сервисов в режиме отладки..."
	docker-compose -f docker-compose.debug.yml up -d
	@echo "⏳ Ожидание готовности сервисов (30 секунд)..."
	@sleep 30
	@echo "📋 Логи OAuth2 сервера:"
	@docker-compose -f docker-compose.debug.yml logs oauth2-server
	@echo ""
	@echo "🔍 Запуск диагностики..."
	@make diagnose-debug

up-simple: stop-conflicts docker-build-simple ## 🚀 Запустить простую версию
	@echo "🚀 Запуск простой версии сервисов..."
	docker-compose -f docker-compose.simple.yml up -d
	@echo "⏳ Ожидание готовности сервисов (20 секунд)..."
	@sleep 20
	@echo "📋 Логи OAuth2 сервера:"
	@docker-compose -f docker-compose.simple.yml logs oauth2-server
	@echo ""
	@echo "🔍 Проверка health endpoint:"
	@curl -s http://localhost:8080/health || echo "❌ Health endpoint недоступен"

up-no-build: stop-conflicts ## 🚀 Запустить без пересборки
	@echo "🚀 Запуск сервисов без пересборки..."
	docker-compose up -d
	@echo "⏳ Ожидание готовности сервисов (60 секунд)..."
	@sleep 60
	@make status

down: ## ⏹️ Остановить все сервисы
	@echo "⏹️  Остановка сервисов..."
	docker-compose down
	docker-compose -f docker-compose.debug.yml down
	docker-compose -f docker-compose.simple.yml down
	@echo "✅ Сервисы остановлены"

logs: ## 📋 Показать логи всех сервисов
	docker-compose logs -f

logs-server: ## 📋 Показать логи OAuth2 сервера
	docker-compose logs -f oauth2-server

logs-db: ## 📋 Показать логи PostgreSQL
	docker-compose logs -f postgres

logs-redis: ## 📋 Показать логи Redis
	docker-compose logs -f redis

status: ## 📊 Показать статус сервисов
	@echo "📊 Статус сервисов:"
	docker-compose ps
	@echo ""
	@echo "🏥 Health Check статусы:"
	@docker inspect oauth2-postgres --format='PostgreSQL: {{.State.Health.Status}}' 2>/dev/null || echo "PostgreSQL: unknown"
	@docker inspect oauth2-redis --format='Redis: {{.State.Health.Status}}' 2>/dev/null || echo "Redis: unknown"
	@docker inspect oauth2-server --format='OAuth2 Server: {{.State.Health.Status}}' 2>/dev/null || echo "OAuth2 Server: unknown"

restart: ## 🔄 Перезапустить все сервисы
	@echo "🔄 Перезапуск сервисов..."
	docker-compose restart
	@echo "✅ Сервисы перезапущены"

restart-server: ## 🔄 Перезапустить только OAuth2 сервер
	@echo "🔄 Перезапуск OAuth2 сервера..."
	docker-compose restart oauth2-server
	@echo "✅ OAuth2 сервер перезапущен"

check-ports: ## 🔌 Проверить занятые порты
	@echo "🔍 Проверка портов:"
	@echo "Порт 5433 (PostgreSQL):"
	@nc -z localhost 5433 && echo "  ✅ Доступен" || echo "  ❌ Недоступен"
	@echo "Порт 6380 (Redis):"
	@nc -z localhost 6380 && echo "  ✅ Доступен" || echo "  ❌ Недоступен"
	@echo "Порт 8080 (OAuth2):"
	@nc -z localhost 8080 && echo "  ✅ Доступен" || echo "  ❌ Недоступен"

shell: ## 🐚 Подключиться к OAuth2 серверу
	docker-compose exec oauth2-server sh

db-shell: ## 🐚 Подключиться к PostgreSQL
	docker-compose exec postgres psql -U oauth2_user -d oauth2_db

redis-shell: ## 🐚 Подключиться к Redis
	docker-compose exec redis redis-cli -a redis_password

docker-test: ## 🧪 Запустить тесты в Docker
	docker-compose exec oauth2-server go test ./... -v

diagnose: ## 🔍 Полная диагностика системы
	@chmod +x scripts/diagnose.sh
	@./scripts/diagnose.sh

diagnose-container: ## 🔍 Детальная диагностика контейнера
	@chmod +x scripts/container-debug.sh
	@./scripts/container-debug.sh

diagnose-debug: ## 🔍 Диагностика debug версии
	@echo "🔍 Диагностика debug версии..."
	@docker-compose -f docker-compose.debug.yml ps
	@echo ""
	@echo "📋 Логи OAuth2 Server (debug):"
	@docker-compose -f docker-compose.debug.yml logs --tail=50 oauth2-server

health: ## 🏥 Проверить health endpoint
	@echo "🏥 Проверка health endpoint:"
	@curl -s http://localhost:8080/health | jq . || curl -s http://localhost:8080/health || echo "❌ Health endpoint недоступен"

# ==================== БЫСТРЫЕ КОМАНДЫ ====================

quick-start: ## 🚀 Быстрый старт
	@echo "🚀 Быстрый старт OAuth2 сервера..."
	@if [ ! -f .env ]; then cp .env.example .env; echo "📝 Создан .env файл"; fi
	@make up
	@echo ""
	@echo "✅ Сервер должен быть запущен!"
	@echo "🔍 Запуск диагностики..."
	@make diagnose
	@echo ""
	@echo "🌐 Доступные URL:"
	@echo "   OAuth2 Server: http://localhost:8080"
	@echo "   Health Check:  http://localhost:8080/health"
	@echo "   PostgreSQL:    localhost:5433"
	@echo "   Redis:         localhost:6380"

quick-start-simple: ## 🚀 Быстрый старт простой версии
	@echo "🚀 Быстрый старт простой версии OAuth2 сервера..."
	@make up-simple
	@echo ""
	@echo "🌐 Доступные URL:"
	@echo "   OAuth2 Server: http://localhost:8080"
	@echo "   PostgreSQL:    localhost:5433"
	@echo "   Redis:         localhost:6380"

debug: ## 🐛 Режим отладки
	@echo "🐛 Запуск в режиме отладки..."
	@make down
	@LOG_LEVEL=debug docker-compose up --build

dev: ## 👨‍💻 Режим разработки (локальная сборка + Docker БД)
	@echo "👨‍💻 Запуск в режиме разработки..."
	@make stop-conflicts
	@docker-compose up -d postgres redis
	@echo "⏳ Ожидание готовности БД..."
	@sleep 10
	@echo "🔨 Локальная сборка..."
	@make build
	@echo "🚀 Запуск локального сервера..."
	@DATABASE_URL="postgres://oauth2_user:oauth2_password@localhost:5433/oauth2_db?sslmode=disable" \
	 REDIS_URL="redis://:redis_password@localhost:6380/0" \
	 ./go_oauth2_server

help: ## 📚 Показать справку
	@echo "OAuth2 Server - Команды разработки и развертывания"
	@echo ""
	@echo "🔧 РАЗРАБОТКА:"
	@grep -E '^[a-zA-Z_-]+:.*?## 🛠|^[a-zA-Z_-]+:.*?## 📦|^[a-zA-Z_-]+:.*?## ⚙️|^[a-zA-Z_-]+:.*?## 🧹|^[a-zA-Z_-]+:.*?## 🧪|^[a-zA-Z_-]+:.*?## 🧼|^[a-zA-Z_-]+:.*?## 🌐' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ""
	@echo "🐳 DOCKER:"
	@grep -E '^[a-zA-Z_-]+:.*?## 🧹|^[a-zA-Z_-]+:.*?## 🛑|^[a-zA-Z_-]+:.*?## 🔨|^[a-zA-Z_-]+:.*?## 🚀|^[a-zA-Z_-]+:.*?## ⏹️|^[a-zA-Z_-]+:.*?## 📋|^[a-zA-Z_-]+:.*?## 📊|^[a-zA-Z_-]+:.*?## 🔄|^[a-zA-Z_-]+:.*?## 🔌|^[a-zA-Z_-]+:.*?## 🐚|^[a-zA-Z_-]+:.*?## 🔍|^[a-zA-Z_-]+:.*?## 🏥' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ""
	@echo "⚡ БЫСТРЫЕ КОМАНДЫ:"
	@grep -E '^[a-zA-Z_-]+:.*?## 🚀|^[a-zA-Z_-]+:.*?## 🐛|^[a-zA-Z_-]+:.*?## 👨‍💻|^[a-zA-Z_-]+:.*?## 📚' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ""
	@echo "Примеры использования:"
	@echo "  make quick-start-simple  # Простая версия для отладки"
	@echo "  make diagnose-container  # Детальная диагностика контейнера"
	@echo "  make logs-server         # Логи OAuth2 сервера"
	@echo "  make dev                 # Локальная разработка"

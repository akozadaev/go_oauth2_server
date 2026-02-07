# Makefile для OAuth2 сервера
.PHONY: help tools generate build release fmt test test-coverage lint-full lint-fix check clean-all clean-deps clean-deps-safe fix-network vendor stop-conflicts docker-build docker-build-simple docker-build-offline up up-simple up-no-build down logs logs-server logs-db logs-redis logs-fixed status restart restart-server check-ports shell db-shell redis-shell shell-fixed docker-test diagnose diagnose-container health quick-start quick-start-simple quick-start-fixed debug dev clean-tokens show-tokens count-tokens fix-perms test-unit test-integration test-docker test-docker-compose

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



docker-build-simple: clean-all ## 🔨 Собрать простые Docker образы
	@echo "🔨 Сборка простых Docker образов..."
	docker-compose -f docker-compose.simple.yml build --no-cache --force-rm
	@echo "✅ Простые Docker образы собраны"

docker-build-fixed: clean-all clean-deps ## 🔨 Собрать исправленные Docker образы
	@echo "🔨 Сборка исправленных Docker образов..."
	docker build -f Dockerfile.fixed -t oauth2-server:fixed --no-cache .
	@echo "✅ Исправленные Docker образы собраны"

up: stop-conflicts docker-build ## 🚀 Запустить все сервисы
	@echo "🚀 Запуск сервисов..."
	docker-compose up -d
	@echo "⏳ Ожидание готовности сервисов (60 секунд)..."
	@sleep 60
	@make status
	@echo "✅ Сервисы запущены"



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

up-fixed: stop-conflicts docker-build-fixed ## 🚀 Запустить исправленную версию
	@echo "🚀 Запуск исправленной версии сервисов..."
	docker run -d --name oauth2-server-fixed \
		-p 8080:8080 \
		-e PORT=8080 \
		-e DATABASE_URL="postgres://oauth2_user:oauth2_password@host.docker.internal:5433/oauth2_db?sslmode=disable" \
		-e JWT_SECRET="your-super-secret-jwt-key-change-this-in-production-make-it-at-least-32-characters-long" \
		-e LOG_LEVEL=debug \
		oauth2-server:fixed
	@echo "⏳ Ожидание готовности сервиса (20 секунд)..."
	@sleep 20
	@echo "📋 Логи исправленной версии:"
	@docker logs oauth2-server-fixed
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

logs-fixed: ## 📋 Показать логи исправленной версии
	docker logs -f oauth2-server-fixed

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

shell-fixed: ## 🐚 Подключиться к исправленной версии
	docker exec -it oauth2-server-fixed sh

docker-test: ## 🧪 Запустить тесты в Docker
	docker-compose exec oauth2-server go test ./... -v

diagnose: ## 🔍 Полная диагностика системы
	@chmod +x scripts/diagnose.sh
	@./scripts/diagnose.sh

diagnose-container: ## 🔍 Детальная диагностика контейнера
	@chmod +x scripts/container-debug.sh
	@./scripts/container-debug.sh



health: ## 🏥 Проверить health endpoint
	@echo "🏥 Проверка health endpoint:"
	@curl -s http://localhost:8080/health | jq . || curl -s http://localhost:8080/health || echo "❌ Health endpoint недоступен"

doc: ## 📚 Генерация Swagger-документации
	go install github.com/swaggo/swag/cmd/swag@latest
	swag init -g cmd/server/main.go --dir . --pd --parseGoList=false --parseDepth=2 -o ./docs

# ==================== ТОКЕНЫ ====================

clean-tokens: ## 🧹 Очистить истекшие токены
	@echo "🧹 Очистка истекших токенов..."
	@chmod +x scripts/cleanup-tokens.sh
	@./scripts/cleanup-tokens.sh

show-tokens: ## 📊 Показать активные токены
	@echo "📊 Активные токены:"
	@docker-compose exec postgres psql -U oauth2_user -d oauth2_db -c "\
		SELECT client_id, user_id, scope, \
		       access_expires_at, refresh_expires_at, \
		       created_at \
		FROM oauth2_tokens \
		WHERE access_expires_at > NOW() \
		ORDER BY created_at DESC \
		LIMIT 10;"

count-tokens: ## 📈 Подсчет токенов
	@echo "📈 Статистика токенов:"
	@docker-compose exec postgres psql -U oauth2_user -d oauth2_db -c "\
		SELECT \
		    COUNT(*) as total_tokens, \
		    COUNT(CASE WHEN access_expires_at > NOW() THEN 1 END) as active_tokens, \
		    COUNT(CASE WHEN access_expires_at <= NOW() THEN 1 END) as expired_tokens \
		FROM oauth2_tokens;"

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

quick-start-fixed: ## 🚀 Быстрый старт исправленной версии
	@echo "🚀 Быстрый старт исправленной версии OAuth2 сервера..."
	@make up-fixed
	@echo ""
	@echo "🌐 Доступные URL:"
	@echo "   OAuth2 Server: http://localhost:8080"
	@echo "   Health Check:  http://localhost:8080/health"

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

fix-perms:
	@echo "Установка прав доступа для папок и фалов кода..."
	@# Права 644 для всех .go, .mod, .sum и других исходных файлов
	find . -type f \( -name "*.go" -o -name "go.mod" -o -name "go.sum" -o -name "*.yaml" -o -name "*.yml" -o -name "*.json" -o -name "*.toml" \) -exec chmod 644 {} + 2>/dev/null || true
	@# Права 755 для всех директорий
	find . -type d -exec chmod 755 {} + 2>/dev/null || true
	@echo "Права установлены."

help: ## 📚 Показать справку
	@echo "OAuth2 Server - Команды разработки и развертывания"
	@echo ""
	@echo "🔧 РАЗРАБОТКА:"
	@grep -E '^[a-zA-Z_-]+:.*?## 🛠|^[a-zA-Z_-]+:.*?## 📦|^[a-zA-Z_-]+:.*?## ⚙️|^[a-zA-Z_-]+:.*?## 🧹|^[a-zA-Z_-]+:.*?## 🧪|^[a-zA-Z_-]+:.*?## 🧼|^[a-zA-Z_-]+:.*?## 🌐' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ""
	@echo "🐳 DOCKER:"
	@grep -E '^[a-zA-Z_-]+:.*?## 🧹|^[a-zA-Z_-]+:.*?## 🛑|^[a-zA-Z_-]+:.*?## 🔨|^[a-zA-Z_-]+:.*?## 🚀|^[a-zA-Z_-]+:.*?## ⏹️|^[a-zA-Z_-]+:.*?## 📋|^[a-zA-Z_-]+:.*?## 📊|^[a-zA-Z_-]+:.*?## 🔄|^[a-zA-Z_-]+:.*?## 🔌|^[a-zA-Z_-]+:.*?## 🐚|^[a-zA-Z_-]+:.*?## 🔍|^[a-zA-Z_-]+:.*?## 🏥' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ""
	@echo "🔑 ТОКЕНЫ:"
	@grep -E '^[a-zA-Z_-]+:.*?## 🧹|^[a-zA-Z_-]+:.*?## 📊|^[a-zA-Z_-]+:.*?## 📈' $(MAKEFILE_LIST) | grep -E 'tokens|clean-tokens|show-tokens|count-tokens' | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ""
	@echo "⚡ БЫСТРЫЕ КОМАНДЫ:"
	@grep -E '^[a-zA-Z_-]+:.*?## 🚀|^[a-zA-Z_-]+:.*?## 🐛|^[a-zA-Z_-]+:.*?## 👨‍💻|^[a-zA-Z_-]+:.*?## 📚' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ""
	@echo "Примеры использования:"
	@echo "  make show-tokens     # Показать активные токены"
	@echo "  make clean-tokens    # Очистить истекшие токены"
	@echo "  make count-tokens    # Статистика токенов"
test-unit: ## 🧪 Запустить только unit тесты
	go test -v ./internal/...

test-integration: ## 🧪 Запустить только integration тесты
	go test -v ./tests/...

test-docker: ## 🧪 Запустить тесты с Docker контейнерами
	./scripts/test-with-docker.sh

test-docker-compose: ## 🧪 Запустить тесты с Docker Compose
	docker-compose -f docker-compose.test.yml up --build --abort-on-container-exit --exit-code-from tests

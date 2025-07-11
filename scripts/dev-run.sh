#!/bin/bash

# Скрипт для запуска в режиме разработки

set -e

echo "👨‍💻 Запуск OAuth2 сервера в режиме разработки..."

# Проверяем .env файл
if [ ! -f .env ]; then
    echo "📝 Создание .env файла..."
    cat > .env << EOF
# Разработка
PORT=8080
LOG_LEVEL=debug

# База данных (Docker)
DATABASE_URL=postgres://oauth2_user:oauth2_password@localhost:5433/oauth2_db?sslmode=disable

# JWT
JWT_SECRET=dev-secret-key-change-in-production-at-least-32-chars-long

# Токены
TOKEN_EXPIRATION_MINUTES=60
REFRESH_EXPIRATION_HOURS=168
EOF
    echo "✅ .env файл создан"
fi

# Запускаем только БД в Docker
echo "🐳 Запуск PostgreSQL в Docker..."
docker-compose up -d postgres

echo "⏳ Ожидание готовности PostgreSQL (15 секунд)..."
sleep 15

# Проверяем подключение к БД
echo "🔍 Проверка подключения к PostgreSQL..."
if nc -z localhost 5433; then
    echo "✅ PostgreSQL доступен"
else
    echo "❌ PostgreSQL недоступен"
    echo "📋 Логи PostgreSQL:"
    docker-compose logs postgres
    exit 1
fi

# Устанавливаем переменные окружения для разработки
export PORT=8080
export DATABASE_URL="postgres://oauth2_user:oauth2_password@localhost:5433/oauth2_db?sslmode=disable"
export JWT_SECRET="dev-secret-key-change-in-production-at-least-32-chars-long"
export TOKEN_EXPIRATION_MINUTES=60
export REFRESH_EXPIRATION_HOURS=168
export LOG_LEVEL=debug

echo "🚀 Запуск OAuth2 сервера..."
echo "🌐 Сервер будет доступен на http://localhost:8080"
echo "🏥 Health check: http://localhost:8080/health"
echo ""
echo "Для остановки нажмите Ctrl+C"
echo ""

# Запускаем сервер
go run ./cmd/server/main.go

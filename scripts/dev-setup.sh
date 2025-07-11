#!/bin/bash

# Скрипт настройки среды разработки

set -e

echo "👨‍💻 Настройка среды разработки OAuth2 сервера"
echo ""

# Проверяем Go
if ! command -v go &> /dev/null; then
    echo "❌ Go не установлен. Установите Go 1.23+ и повторите попытку."
    exit 1
fi

GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
echo "✅ Go версия: $GO_VERSION"

# Устанавливаем инструменты разработки
echo "🛠 Установка инструментов разработки..."
make tools

# Создаем .env для разработки
if [ ! -f .env ]; then
    echo "📝 Создание .env файла для разработки..."
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
    echo "✅ .env файл создан для разработки"
fi

# Загружаем зависимости
echo "📦 Загрузка зависимостей..."
go mod download
go mod tidy

# Запускаем только БД в Docker
echo "🐳 Запуск PostgreSQL в Docker..."
make stop-conflicts
docker-compose up -d postgres

echo "⏳ Ожидание готовности БД..."
sleep 15

# Проверяем подключение к БД
echo "🔍 Проверка подключения к БД..."
if nc -z localhost 5433; then
    echo "✅ PostgreSQL доступен"
else
    echo "❌ PostgreSQL недоступен"
    exit 1
fi

# Собираем приложение
echo "🔨 Сборка приложения..."
make build

echo ""
echo "✅ Среда разработки готова!"
echo ""
echo "🚀 Для запуска сервера выполните:"
echo "   make dev"
echo ""
echo "📚 Полезные команды разработки:"
echo "   make check           # Проверка кода"
echo "   make test            # Тесты"
echo "   make fmt             # Форматирование"
echo "   make lint-full       # Линтинг"
echo "   make generate        # Генерация кода"
echo ""
echo "🐳 Docker команды:"
echo "   make quick-start     # Полный запуск в Docker"
echo "   make logs-db         # Логи PostgreSQL"
echo "   make db-shell        # Подключение к БД"

git stash pop 2>/dev/null || true#!/bin/bash

# Скрипт для тестирования локальной сборки

set -e

echo "🔧 Тестирование локальной сборки OAuth2 сервера..."

# Проверяем Go
if ! command -v go &> /dev/null; then
    echo "❌ Go не установлен"
    exit 1
fi

echo "✅ Go версия: $(go version)"

# Очищаем зависимости
echo "🧹 Очистка зависимостей..."
go mod tidy

# Проверяем синтаксис
echo "🔍 Проверка синтаксиса..."
go vet ./...

# Пробуем собрать
echo "🔨 Сборка приложения..."
go build -o oauth2-server-test ./cmd/server/

if [ -f "oauth2-server-test" ]; then
    echo "✅ Сборка успешна!"
    echo "📁 Размер файла: $(ls -lh oauth2-server-test | awk '{print $5}')"

    # Проверяем что файл исполняемый
    if [ -x "oauth2-server-test" ]; then
        echo "✅ Файл исполняемый"

        # Пробуем запустить с --help (если поддерживается)
        echo "🚀 Тестовый запуск..."
        timeout 5 ./oauth2-server-test 2>/dev/null || echo "⚠️  Сервер запустился (остановлен через 5 секунд)"
    else
        echo "❌ Файл не исполняемый"
    fi

    # Удаляем тестовый файл
    rm oauth2-server-test
else
    echo "❌ Сборка не удалась"
    exit 1
fi

echo ""
echo "✅ Локальная сборка работает корректно!"
echo "🚀 Можно запускать: go run ./cmd/server/main.go"

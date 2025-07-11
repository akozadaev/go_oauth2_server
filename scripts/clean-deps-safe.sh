#!/bin/bash

# Безопасная очистка зависимостей без root прав

set -e

echo "🧹 Безопасная очистка зависимостей Go..."

# Удаляем vendor если есть
if [ -d "vendor" ]; then
    echo "🗑️  Удаление vendor директории..."
    rm -rf vendor/
fi

# Проверяем права на кеш модулей
MODCACHE=$(go env GOMODCACHE)
if [ -w "$MODCACHE" ] 2>/dev/null; then
    echo "🧹 Очистка модульного кеша..."
    go clean -modcache
else
    echo "⚠️  Нет прав для очистки глобального кеша модулей"
    echo "💡 Используем локальную очистку..."

    # Создаем временный GOMODCACHE в проекте
    export GOMODCACHE="$(pwd)/.modcache"
    mkdir -p "$GOMODCACHE"
    echo "📁 Создан локальный кеш модулей: $GOMODCACHE"
fi

# Обновляем go.mod и go.sum
echo "📦 Обновление зависимостей..."
go mod tidy

# Проверяем зависимости
echo "🔍 Проверка зависимостей..."
go mod verify || echo "⚠️  Некоторые зависимости могут быть недоступны"

# Пробуем загрузить зависимости
echo "⬇️  Загрузка зависимостей..."
go mod download || echo "⚠️  Некоторые зависимости не удалось загрузить"

echo "✅ Безопасная очистка завершена!"

#!/bin/bash

# Скрипт для запуска тестов
# Тесты используют dockertest для автоматического запуска PostgreSQL в Docker.
# Убедитесь, что Docker запущен: docker info

set -e

echo "Запуск тестов (dockertest автоматически поднимет PostgreSQL)..."

if ! docker info > /dev/null 2>&1; then
    echo "❌ Docker не запущен. Запустите Docker и попробуйте снова."
    exit 1
fi

go test -v ./...

echo "✅ Тесты завершены!"
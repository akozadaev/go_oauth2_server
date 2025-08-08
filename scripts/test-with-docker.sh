#!/bin/bash

# Скрипт для запуска тестов с Docker

set -e

echo "Запуск тестов с Docker..."

# Проверяем, что Docker запущен
if ! docker info > /dev/null 2>&1; then
    echo "❌ Docker не запущен. Пожалуйста, запустите Docker и попробуйте снова."
    exit 1
fi

# Очищаем старые контейнеры
echo "Очистка старых контейнеров..."
docker stop test-postgres 2>/dev/null || true
docker rm test-postgres 2>/dev/null || true

# Создаем тестовую сеть
echo "Создание тестовой сети..."
docker network create test-network 2>/dev/null || true

# Запускаем PostgreSQL контейнер
echo "Запуск PostgreSQL контейнера..."
docker run -d \
    --name test-postgres \
    --network test-network \
    -e POSTGRES_DB=test_db \
    -e POSTGRES_USER=test_user \
    -e POSTGRES_PASSWORD=test_password \
    -p 5433:5432 \
    postgres:15-alpine

# Ждем пока PostgreSQL будет готов
echo "Ожидание готовности PostgreSQL..."
for i in {1..30}; do
    if docker exec test-postgres pg_isready -U test_user -d test_db > /dev/null 2>&1; then
        echo "✅ PostgreSQL готов!"
        break
    fi
    echo "Попытка $i/30..."
    sleep 2
done

# Проверяем, что PostgreSQL действительно готов
if ! docker exec test-postgres pg_isready -U test_user -d test_db > /dev/null 2>&1; then
    echo "PostgreSQL не готов после 30 попыток"
    docker logs test-postgres
    docker stop test-postgres
    docker rm test-postgres
    exit 1
fi

# Устанавливаем переменную окружения для тестов
export TEST_DATABASE_URL="postgres://test_user:test_password@localhost:5433/test_db?sslmode=disable"

echo "Запуск тестов..."
go test -v ./...

# Очистка
echo "Очистка..."
docker stop test-postgres
docker rm test-postgres

echo "Тесты завершены!"
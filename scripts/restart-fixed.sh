#!/bin/bash

# Скрипт для быстрого перезапуска исправленной версии

set -e

echo "🔄 Перезапуск исправленной версии OAuth2 сервера..."

# Остановка и удаление контейнеров
echo "⏹️ Остановка старых контейнеров..."
docker-compose down --remove-orphans 2>/dev/null || true
docker stop oauth2-server oauth2-postgres oauth2-adminer 2>/dev/null || true
docker rm oauth2-server oauth2-postgres oauth2-adminer 2>/dev/null || true

# Пересборка образа
echo "🔨 Пересборка образа..."
docker-compose build --no-cache

# Запуск сервисов
echo "🚀 Запуск сервисов..."
docker-compose up -d

echo "⏳ Ожидание готовности сервисов (30 секунд)..."
sleep 30

# Проверка статуса
echo "📊 Статус контейнеров:"
docker-compose ps

# Проверка подключений
echo ""
echo "🔍 Проверка подключений..."

# Проверяем PostgreSQL
echo "PostgreSQL (5433):"
if nc -z localhost 5433; then
    echo "✅ Доступен"
else
    echo "❌ Недоступен"
    echo "📋 Логи PostgreSQL:"
    docker-compose logs postgres
    exit 1
fi

# Проверяем OAuth2 Server
echo "OAuth2 Server (8080):"
if nc -z localhost 8080; then
    echo "✅ Доступен"
else
    echo "❌ Недоступен"
    echo "📋 Логи OAuth2 Server:"
    docker-compose logs oauth2-server
    exit 1
fi

# Проверка health endpoint
echo "🏥 Проверка health endpoint:"
if curl -s http://localhost:8080/health > /dev/null; then
    echo "✅ Health endpoint работает"
    echo "📊 Ответ health endpoint:"
    curl -s http://localhost:8080/health | head -3
else
    echo "❌ Health endpoint не работает"
    echo "📋 Логи OAuth2 Server:"
    docker-compose logs oauth2-server
    exit 1
fi

echo ""
echo "✅ Перезапуск завершен!"
echo ""
echo "🌐 Доступные URL:"
echo "   OAuth2 Server: http://localhost:8080"
echo "   Health Check:  http://localhost:8080/health"
echo "   PostgreSQL:    localhost:5433"
echo "   Adminer:       http://localhost:8081"
echo ""
echo "📚 Полезные команды:"
echo "   docker-compose logs        - просмотр логов"
echo "   docker-compose ps          - статус контейнеров"
echo "   ./scripts/diagnose.sh      - диагностика"

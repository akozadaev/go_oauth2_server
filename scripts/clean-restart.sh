#!/bin/bash

# Скрипт для полной очистки и перезапуска

set -e

echo "🧹 Полная очистка и перезапуск OAuth2 сервера..."
echo ""

# Остановка всех контейнеров
echo "⏹️ Остановка всех контейнеров..."
docker-compose down --remove-orphans --volumes 2>/dev/null || true
docker stop oauth2-server oauth2-postgres oauth2-adminer 2>/dev/null || true
docker rm oauth2-server oauth2-postgres oauth2-adminer 2>/dev/null || true

# Удаление volumes (опционально)
read -p "🗑️ Удалить все данные PostgreSQL? (y/N): " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo "🗑️ Удаление volumes..."
    docker volume rm go_oauth2_server_postgres_data 2>/dev/null || true
    echo "✅ Volumes удалены"
fi

# Очистка образов (опционально)
read -p "🧹 Удалить старые образы? (y/N): " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo "🧹 Удаление старых образов..."
    docker rmi go_oauth2_server_oauth2-server 2>/dev/null || true
    echo "✅ Образы удалены"
fi

# Пересборка образов
echo "🔨 Пересборка образов..."
docker-compose build --no-cache

# Запуск только PostgreSQL сначала
echo "🚀 Запуск PostgreSQL..."
docker-compose up -d postgres

echo "⏳ Ожидание готовности PostgreSQL (30 секунд)..."
sleep 30

# Проверка PostgreSQL
echo "🔍 Проверка PostgreSQL..."
if nc -z localhost 5433; then
    echo "✅ PostgreSQL доступен"
else
    echo "❌ PostgreSQL недоступен"
    echo "📋 Логи PostgreSQL:"
    docker-compose logs postgres
    exit 1
fi

# Проверка подключения к БД
echo "🗄️ Проверка подключения к БД..."
docker-compose exec postgres psql -U oauth2_user -d oauth2_db -c "SELECT version();" 2>/dev/null && echo "✅ Подключение к БД работает" || echo "❌ Проблемы с подключением к БД"

# Запуск OAuth2 Server
echo "🚀 Запуск OAuth2 Server..."
docker-compose up -d oauth2-server

echo "⏳ Ожидание готовности OAuth2 Server (30 секунд)..."
sleep 30

# Проверка OAuth2 Server
echo "🔍 Проверка OAuth2 Server..."
if nc -z localhost 8080; then
    echo "✅ OAuth2 Server доступен"
else
    echo "❌ OAuth2 Server недоступен"
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

# Финальный статус
echo ""
echo "📊 Финальный статус контейнеров:"
docker-compose ps

echo ""
echo "✅ Полная очистка и перезапуск завершены!"
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
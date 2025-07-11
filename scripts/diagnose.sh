#!/bin/bash

# Скрипт диагностики OAuth2 сервера

set -e

echo "🔍 Диагностика OAuth2 сервера..."
echo ""

# Проверяем статус контейнеров
echo "📦 Статус контейнеров:"
docker-compose ps
echo ""

# Проверяем логи каждого сервиса
echo "📋 Логи PostgreSQL (последние 20 строк):"
docker-compose logs --tail=20 postgres
echo ""

echo "📋 Логи OAuth2 Server (последние 50 строк):"
docker-compose logs --tail=50 oauth2-server
echo ""

# Проверяем health check
echo "🏥 Health Check статус:"
docker inspect oauth2-server --format='{{.State.Health.Status}}' 2>/dev/null || echo "Health check недоступен"
echo ""

# Проверяем детали health check
echo "🏥 Детали Health Check:"
docker inspect oauth2-server --format='{{range .State.Health.Log}}{{.Output}}{{end}}' 2>/dev/null || echo "Детали недоступны"
echo ""

# Проверяем сетевое подключение
echo "🌐 Проверка сетевого подключения:"
echo "Postgres -> OAuth2 Server:"
docker-compose exec postgres ping -c 2 oauth2-server 2>/dev/null || echo "❌ Недоступен"
echo ""

# Проверяем порты
echo "🔌 Проверка портов:"
echo "PostgreSQL (5433):"
nc -z localhost 5433 && echo "✅ Доступен" || echo "❌ Недоступен"

echo "OAuth2 Server (8080):"
nc -z localhost 8080 && echo "✅ Доступен" || echo "❌ Недоступен"
echo ""

# Проверяем подключение к БД
echo "🗄️ Проверка подключения к БД:"
docker-compose exec postgres psql -U root -d postgres -c "SELECT version();" 2>/dev/null && echo "✅ Подключение к БД работает" || echo "❌ Проблемы с подключением к БД"
echo ""

# Проверяем health endpoint
echo "🏥 Проверка health endpoint:"
if curl -s http://localhost:8080/health > /dev/null; then
    echo "✅ Health endpoint доступен"
    echo "📊 Ответ health endpoint:"
    curl -s http://localhost:8080/health | head -5
else
    echo "❌ Health endpoint недоступен"
fi
echo ""

# Проверяем переменные окружения
echo "🌍 Переменные окружения OAuth2 Server:"
docker-compose exec oauth2-server env | grep -E "(PORT|DATABASE|JWT|LOG)" 2>/dev/null || echo "❌ Не удается получить переменные окружения"
echo ""

# Проверяем файлы в контейнере
echo "📁 Файлы в контейнере OAuth2 Server:"
docker-compose exec oauth2-server ls -la /app 2>/dev/null || echo "❌ Не удается получить список файлов"
echo ""

# Проверяем права на файлы
echo "🔐 Права на oauth2-server:"
docker-compose exec oauth2-server ls -la /app/oauth2-server 2>/dev/null || echo "❌ Файл oauth2-server не найден"
echo ""

echo "✅ Диагностика завершена!"
echo ""
echo "🔧 Если есть проблемы, попробуйте:"
echo "   ./scripts/troubleshoot.sh"
echo "   docker-compose down && docker-compose up -d"

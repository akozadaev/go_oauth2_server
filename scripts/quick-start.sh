#!/bin/bash

# Скрипт быстрого запуска OAuth2 сервера

set -e

echo "🚀 Быстрый запуск OAuth2 сервера..."
echo ""

# Проверяем необходимые инструменты
echo "🔍 Проверка инструментов..."
for cmd in docker docker-compose curl nc; do
    if ! command -v $cmd &> /dev/null; then
        echo "❌ $cmd не установлен"
        exit 1
    fi
done
echo "✅ Все инструменты установлены"
echo ""

# Останавливаем конфликтующие контейнеры
echo "🛑 Остановка конфликтующих контейнеров..."
docker-compose --profile dev down --remove-orphans 2>/dev/null || true
docker stop oauth2-server oauth2-postgres oauth2-adminer 2>/dev/null || true
docker rm oauth2-server oauth2-postgres oauth2-adminer 2>/dev/null || true
echo "✅ Конфликтующие контейнеры остановлены"
echo ""

# Очищаем старые образы (автоматически, без интерактивного запроса)
echo "🧹 Очистка старых образов..."
docker-compose --profile dev build --no-cache

# Запускаем сервисы
echo "🐳 Запуск сервисов..."
docker-compose --profile dev up -d

echo "⏳ Ожидание готовности сервисов..."
sleep 30

# Проверяем статус контейнеров
echo "📊 Статус контейнеров:"
docker-compose --profile dev ps

echo ""
echo "🔍 Проверка подключений..."

# Проверяем PostgreSQL
echo "PostgreSQL (5433):"
if nc -z localhost 5433; then
    echo "✅ Доступен"
else
    echo "❌ Недоступен"
    echo "📋 Логи PostgreSQL:"
    docker-compose --profile dev logs postgres
    exit 1
fi

# Проверяем OAuth2 Server
echo "OAuth2 Server (8080):"
if nc -z localhost 8080; then
    echo "✅ Доступен"
else
    echo "❌ Недоступен"
    echo "📋 Логи OAuth2 Server:"
    docker-compose --profile dev logs oauth2-server
    exit 1
fi

# Проверяем health endpoint
echo "🏥 Health Check:"
if curl -s http://localhost:8080/health > /dev/null; then
    echo "✅ Health endpoint работает"
else
    echo "❌ Health endpoint не работает"
    echo "📋 Логи OAuth2 Server:"
    docker-compose --profile dev logs oauth2-server
    exit 1
fi

# Проверяем Adminer
echo "Adminer (8081):"
if nc -z localhost 8081; then
    echo "✅ Доступен"
else
    echo "❌ Недоступен"
    echo "📋 Логи Adminer:"
    docker-compose --profile dev logs adminer
fi

echo ""
echo "✅ OAuth2 сервер успешно запущен!"
echo ""
echo "🌐 Доступные URL:"
echo "   OAuth2 Server: http://localhost:8080"
echo "   Health Check:  http://localhost:8080/health"
echo "   PostgreSQL:    localhost:5433"
echo "   Adminer:       http://localhost:8081"
echo ""
echo "📚 Полезные команды:"
echo "   docker-compose --profile dev logs        - просмотр логов"
echo "   docker-compose --profile dev ps          - статус контейнеров"
echo "   docker-compose --profile dev down        - остановка сервисов"
echo "   docker-compose --profile dev restart     - перезапуск сервисов"
echo ""
echo "🔧 Для диагностики проблем:"
echo "   ./scripts/diagnose.sh"
echo "   ./scripts/troubleshoot.sh"

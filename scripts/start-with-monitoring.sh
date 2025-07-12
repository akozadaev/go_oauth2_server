#!/bin/bash

# Скрипт для запуска OAuth2 сервера с мониторингом

set -e

echo "📊 Запуск OAuth2 сервера с мониторингом..."
echo ""

# Остановка старых контейнеров
echo "⏹️ Остановка старых контейнеров..."
docker-compose down --remove-orphans 2>/dev/null || true

# Запуск всех сервисов включая мониторинг
echo "🚀 Запуск сервисов с мониторингом..."
docker-compose --profile dev --profile monitoring up -d

echo "⏳ Ожидание готовности сервисов (45 секунд)..."
sleep 45

# Проверка статуса
echo "📊 Статус контейнеров:"
docker-compose ps

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

# Проверяем Prometheus
echo "Prometheus (9090):"
if nc -z localhost 9090; then
    echo "✅ Доступен"
else
    echo "❌ Недоступен"
    echo "📋 Логи Prometheus:"
    docker-compose logs prometheus
    exit 1
fi

# Проверяем Grafana
echo "Grafana (3000):"
if nc -z localhost 3000; then
    echo "✅ Доступен"
else
    echo "❌ Недоступен"
    echo "📋 Логи Grafana:"
    docker-compose logs grafana
    exit 1
fi

# Проверка health endpoint
echo "🏥 Проверка health endpoint:"
if curl -s http://localhost:8080/health > /dev/null; then
    echo "✅ Health endpoint работает"
else
    echo "❌ Health endpoint не работает"
    exit 1
fi

# Проверка метрик
echo "📈 Проверка метрик:"
if curl -s http://localhost:8080/metrics > /dev/null; then
    echo "✅ Метрики доступны"
else
    echo "❌ Метрики недоступны"
    exit 1
fi

echo ""
echo "✅ OAuth2 сервер с мониторингом успешно запущен!"
echo ""
echo "🌐 Доступные URL:"
echo "   OAuth2 Server: http://localhost:8080"
echo "   Health Check:  http://localhost:8080/health"
echo "   Metrics:       http://localhost:8080/metrics"
echo "   PostgreSQL:    localhost:5433"
echo "   Adminer:       http://localhost:8081"
echo "   Prometheus:    http://localhost:9090"
echo "   Grafana:       http://localhost:3000 (admin/admin)"
echo ""
echo "📚 Полезные команды:"
echo "   docker-compose logs        - просмотр логов"
echo "   docker-compose ps          - статус контейнеров"
echo "   ./scripts/diagnose.sh      - диагностика"
echo ""
echo "📊 Настройка Grafana:"
echo "   1. Откройте http://localhost:3000"
echo "   2. Войдите с admin/admin"
echo "   3. Добавьте источник данных Prometheus: http://prometheus:9090"
echo "   4. Импортируйте дашборды для мониторинга" 
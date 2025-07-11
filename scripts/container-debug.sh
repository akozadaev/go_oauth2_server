#!/bin/bash

# Скрипт отладки контейнера OAuth2 сервера

set -e

CONTAINER_NAME="oauth2-server"

echo "🔍 Отладка контейнера $CONTAINER_NAME..."
echo ""

# Проверяем, что контейнер существует
if ! docker ps -a --format "{{.Names}}" | grep -q "^$CONTAINER_NAME$"; then
    echo "❌ Контейнер $CONTAINER_NAME не найден"
    echo "Доступные контейнеры:"
    docker ps -a --format "table {{.Names}}\t{{.Status}}\t{{.Image}}"
    exit 1
fi

# Проверяем статус контейнера
echo "📊 Статус контейнера:"
docker ps --filter "name=$CONTAINER_NAME" --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"
echo ""

# Проверяем логи
echo "📋 Последние логи (50 строк):"
docker logs --tail=50 $CONTAINER_NAME
echo ""

# Проверяем health check
echo "🏥 Health Check:"
docker inspect $CONTAINER_NAME --format='{{.State.Health.Status}}' 2>/dev/null || echo "Health check недоступен"
echo ""

# Проверяем детали health check
echo "🏥 Детали Health Check:"
docker inspect $CONTAINER_NAME --format='{{range .State.Health.Log}}{{.Output}}{{end}}' 2>/dev/null || echo "Детали недоступны"
echo ""

# Проверяем права на файлы
echo "🔐 Права на oauth2-server:"
docker exec $CONTAINER_NAME ls -la /app/oauth2-server 2>/dev/null || echo "❌ Файл oauth2-server не найден"
echo ""

# Пробуем запустить приложение вручную
echo "🚀 Попытка ручного запуска приложения:"
docker exec $CONTAINER_NAME /app/oauth2-server --help 2>/dev/null || echo "❌ Не удается запустить приложение"
echo ""

# Проверяем переменные окружения
echo "🌍 Переменные окружения:"
docker exec $CONTAINER_NAME env | grep -E "(PORT|DATABASE|JWT|LOG)" 2>/dev/null || echo "❌ Не удается получить переменные окружения"
echo ""

# Проверяем сетевые подключения
echo "🌐 Сетевые подключения:"
docker exec $CONTAINER_NAME netstat -tlnp 2>/dev/null || echo "❌ netstat недоступен"
echo ""

# Проверяем доступность БД из контейнера
echo "🔍 Проверка подключения к БД:"
docker exec $CONTAINER_NAME nc -z postgres 5432 2>/dev/null && echo "✅ PostgreSQL доступен" || echo "❌ PostgreSQL недоступен"
echo ""

echo "🔍 Детальная информация о контейнере:"
docker inspect $CONTAINER_NAME --format='{{.State}}' 2>/dev/null || echo "❌ Не удается получить информацию о контейнере"
echo ""

# Проверяем health endpoint
echo "🏥 Проверка health endpoint изнутри контейнера:"
docker exec $CONTAINER_NAME curl -s http://localhost:8080/health 2>/dev/null && echo "✅ Health endpoint работает" || echo "❌ Health endpoint не работает"
echo ""

echo "✅ Отладка завершена!"
echo ""
echo "🔧 Рекомендации:"
echo "   - Если контейнер не запускается, проверьте логи выше"
echo "   - Если проблемы с БД, убедитесь что PostgreSQL запущен"
echo "   - Если проблемы с правами, пересоберите образ"

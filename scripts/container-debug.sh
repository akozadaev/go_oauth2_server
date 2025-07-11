#!/bin/bash

# Скрипт для детальной отладки контейнера

set -e

echo "🔍 Детальная отладка OAuth2 контейнера..."
echo ""

# Определяем какой контейнер использовать
CONTAINER_NAME=""
if docker ps -q --filter "name=oauth2-server-simple" | grep -q .; then
    CONTAINER_NAME="oauth2-server-simple"
    echo "📦 Найден контейнер: oauth2-server-simple"
elif docker ps -q --filter "name=oauth2-server-debug" | grep -q .; then
    CONTAINER_NAME="oauth2-server-debug"
    echo "📦 Найден контейнер: oauth2-server-debug"
elif docker ps -q --filter "name=oauth2-server" | grep -q .; then
    CONTAINER_NAME="oauth2-server"
    echo "📦 Найден контейнер: oauth2-server"
else
    echo "❌ Не найден ни один OAuth2 контейнер"
    echo "📦 Все контейнеры:"
    docker ps -a --filter "name=oauth2"
    exit 1
fi

echo ""

# Проверяем статус контейнера
echo "📦 Статус контейнера $CONTAINER_NAME:"
docker ps -a --filter "name=$CONTAINER_NAME" --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"
echo ""

# Проверяем логи контейнера
echo "📋 Полные логи $CONTAINER_NAME:"
docker logs $CONTAINER_NAME 2>&1 || echo "❌ Не удается получить логи"
echo ""

# Проверяем, запущен ли процесс внутри контейнера
echo "🔍 Процессы внутри контейнера:"
docker exec $CONTAINER_NAME ps aux 2>/dev/null || echo "❌ Контейнер не отвечает"
echo ""

# Проверяем файловую систему контейнера
echo "📁 Содержимое /app в контейнере:"
docker exec $CONTAINER_NAME ls -la /app 2>/dev/null || echo "❌ Не удается получить список файлов"
echo ""

# Проверяем права на файлы
echo "🔐 Права на oauth2-server:"
docker exec $CONTAINER_NAME ls -la /app/oauth2-server 2>/dev/null || echo "❌ Файл oauth2-server не найден"
echo ""

# Пробуем запустить приложение вручную
echo "🚀 Попытка ручного зап��ска приложения:"
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

# Проверяем доступность Redis из контейнера
echo "🔍 Проверка подключения к Redis:"
docker exec $CONTAINER_NAME nc -z redis 6379 2>/dev/null && echo "✅ Redis доступен" || echo "❌ Redis недоступен"
echo ""

echo "🔍 Детальная информация о контейнере:"
docker inspect $CONTAINER_NAME --format='{{.State}}' 2>/dev/null || echo "❌ Не удается получить информацию о контейнере"
echo ""

# Проверяем health endpoint
echo "🏥 Проверка health endpoint изнутри контейнера:"
docker exec $CONTAINER_NAME curl -s http://localhost:8080/health 2>/dev/null || echo "❌ Health endpoint недоступен"
echo ""

# Проверяем архитектуру
echo "🏗️ Архитектура контейнера:"
docker exec $CONTAINER_NAME uname -a 2>/dev/null || echo "❌ Не удается получить информацию об архитектуре"

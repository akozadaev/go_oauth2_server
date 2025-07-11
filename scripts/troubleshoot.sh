#!/bin/bash

# Скрипт для диагностики и решения проблем

set -e

echo "🔍 Диагностика OAuth2 сервера..."

# Функция для проверки порта
check_port() {
    local port=$1
    local service=$2
    echo -n "Проверка порта $port ($service): "
    if lsof -i :$port > /dev/null 2>&1; then
        echo "❌ ЗАНЯТ"
        echo "  Процессы на порту $port:"
        lsof -i :$port | head -5
        return 1
    else
        echo "✅ СВОБОДЕН"
        return 0
    fi
}

# Функция для остановки процесса на порту
kill_port() {
    local port=$1
    echo "🔪 Остановка процессов на порту $port..."
    lsof -ti :$port | xargs kill -9 2>/dev/null || true
    sleep 2
}

echo ""
echo "📋 Проверка портов..."

# Проверяем основные порты
PORTS_OK=true

if ! check_port 5433 "PostgreSQL"; then
    PORTS_OK=false
    read -p "Остановить процессы на порту 5433? (y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        kill_port 5433
    fi
fi

if ! check_port 8080 "OAuth2 Server"; then
    PORTS_OK=false
    read -p "Остановить процессы на порту 8080? (y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        kill_port 8080
    fi
fi

echo ""
echo "🐳 Проверка Docker..."

# Проверяем Docker
if ! command -v docker &> /dev/null; then
    echo "❌ Docker не установлен"
    exit 1
else
    echo "✅ Docker установлен: $(docker --version)"
fi

# Проверяем Docker Compose
if ! command -v docker-compose &> /dev/null; then
    echo "❌ Docker Compose не установлен"
    exit 1
else
    echo "✅ Docker Compose установлен: $(docker-compose --version)"
fi

# Останавливаем старые контейнеры
echo ""
echo "🛑 Остановка старых контейнеров..."
docker-compose down --remove-orphans 2>/dev/null || true

echo ""
echo "🔍 Проверка файлов..."

# Проверяем Dockerfile
if [ ! -f "Dockerfile" ]; then
    echo "❌ Отсутствует Dockerfile"
    exit 1
else
    echo "✅ Dockerfile найден"
fi

# Проверяем docker-compose.yml
if [ ! -f "docker-compose.yml" ]; then
    echo "❌ Отсутствует docker-compose.yml"
    exit 1
else
    echo "✅ docker-compose.yml найден"
fi

echo ""
echo "🔨 Пересборка образов..."

# Пересобираем образы
docker-compose build --no-cache

echo ""
echo "🚀 Запуск сервисов..."

# Запускаем сервисы
docker-compose up -d

echo ""
echo "⏳ Ожидание готовности сервисов..."
sleep 30

echo ""
echo "📊 Проверка статуса..."

# Проверяем статус контейнеров
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
fi

# Проверяем OAuth2 Server
echo "OAuth2 Server (8080):"
if nc -z localhost 8080; then
    echo "✅ Доступен"
else
    echo "❌ Недоступен"
    echo "📋 Логи OAuth2 Server:"
    docker-compose logs oauth2-server
fi

# Проверяем health endpoint
echo "🏥 Health Check:"
if curl -s http://localhost:8080/health > /dev/null; then
    echo "✅ Health endpoint работает"
else
    echo "❌ Health endpoint не работает"
fi

echo ""
echo "✅ Диагностика завершена!"
echo ""
echo "🔧 Если проблемы остались:"
echo "   - Проверьте логи: docker-compose logs"
echo "   - Перезапустите: docker-compose restart"
echo "   - Полная пересборка: docker-compose down && docker-compose up --build"

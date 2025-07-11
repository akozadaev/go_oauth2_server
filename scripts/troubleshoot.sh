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

if ! check_port 6380 "Redis"; then
    PORTS_OK=false
    read -p "Остановить процессы на порту 6380? (y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        kill_port 6380
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

if ! command -v docker-compose &> /dev/null; then
    echo "❌ Docker Compose не установлен"
    exit 1
else
    echo "✅ Docker Compose установлен: $(docker-compose --version)"
fi

# Проверяем запущенные контейнеры
echo ""
echo "📦 Существующие контейнеры OAuth2:"
docker ps -a --filter "name=oauth2" --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}" || true

echo ""
echo "🧹 Очистка старых контейнеров..."
docker-compose down --remove-orphans 2>/dev/null || true

echo ""
echo "🔧 Проверка конфигурации..."

# Проверяем наличие необходимых файлов
FILES_OK=true

if [ ! -f "docker-compose.yml" ]; then
    echo "❌ Отсутствует docker-compose.yml"
    FILES_OK=false
else
    echo "✅ docker-compose.yml найден"
fi

if [ ! -f "Dockerfile" ]; then
    echo "❌ Отсутствует Dockerfile"
    FILES_OK=false
else
    echo "✅ Dockerfile найден"
fi

if [ ! -f ".env" ]; then
    echo "⚠️  Отсутствует .env файл"
    if [ -f ".env.example" ]; then
        echo "📝 Создание .env из .env.example..."
        cp .env.example .env
        echo "✅ .env файл создан"
    else
        echo "❌ .env.example также отсутствует"
        FILES_OK=false
    fi
else
    echo "✅ .env файл найден"
fi

# Создаем необходимые директории
echo ""
echo "📁 Создание необходимых директорий..."
mkdir -p nginx/ssl
mkdir -p scripts
mkdir -p monitoring

if [ ! -f "nginx/ssl/nginx.crt" ]; then
    echo "🔐 Создание SSL сертификатов..."
    openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
        -keyout nginx/ssl/nginx.key \
        -out nginx/ssl/nginx.crt \
        -subj "/C=US/ST=State/L=City/O=Organization/CN=localhost" \
        2>/dev/null
    echo "✅ SSL сертификаты созданы"
fi

echo ""
if [ "$PORTS_OK" = true ] && [ "$FILES_OK" = true ]; then
    echo "🎉 Все проверки пройдены! Можно запускать сервер:"
    echo "   make up"
else
    echo "⚠️  Обнаружены проблемы. Исправьте их перед запуском."
fi

echo ""
echo "📚 Полезные команды:"
echo "   make check-ports  - проверить порты"
echo "   make clean        - очистить контейнеры"
echo "   make logs         - посмотреть логи"
echo "   make status       - статус сервисов"

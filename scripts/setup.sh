#!/bin/bash

# Скрипт для первоначальной настройки

set -e

echo "🚀 Настройка OAuth2 сервера..."

# Создание .env файла
if [ ! -f .env ]; then
    echo "📝 Создание .env файла..."
    cp .env.example .env
    echo "✅ .env файл создан. Отредактируйте его перед запуском!"
fi

# Создание SSL сертификатов для nginx
if [ ! -d nginx/ssl ]; then
    echo "🔐 Создание SSL сертификатов..."
    mkdir -p nginx/ssl
    openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
        -keyout nginx/ssl/nginx.key \
        -out nginx/ssl/nginx.crt \
        -subj "/C=US/ST=State/L=City/O=Organization/CN=localhost"
    echo "✅ SSL сертификаты созданы"
fi

# Сборка образов
echo "🔨 Сборка Docker образов..."
docker-compose build

echo "✅ Настройка завершена!"
echo ""
echo "Для запуска выполните:"
echo "  make up          # Обычный запуск"
echo "  make up-dev      # Режим разработки"
echo "  make monitoring  # С мониторингом"
echo ""
echo "Полезные команды:"
echo "  make logs        # Просмотр логов"
echo "  make shell       # Подключение к серверу"
echo "  make db-shell    # Подключение к БД"

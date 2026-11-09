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

bash scripts/setup-tls.sh

# Сборка образов
echo "🔨 Сборка Docker образов..."
docker-compose build

echo "✅ Настройка завершена!"
echo ""
echo "Для запуска выполните:"
echo "  make up          # Обычный запуск"
echo "  docker-compose --profile dev up -d  # С Adminer"
echo "  docker-compose --profile monitoring up -d  # С мониторингом"
echo ""
echo "Полезные команды:"
echo "  make logs        # Просмотр логов"
echo "  make shell       # Подключение к серверу"
echo "  make db-shell    # Подключение к БД"

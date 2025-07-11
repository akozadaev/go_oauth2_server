#!/bin/bash

# Скрипт для полного сброса Docker окружения

set -e

echo "🔥 ВНИМАНИЕ: Этот скрипт удалит ВСЕ Docker контейнеры, образы и тома!"
echo "Это может повлиять на другие проекты Docker."
echo ""
read -p "Продолжить? (yes/no): " -r
if [[ ! $REPLY =~ ^yes$ ]]; then
    echo "Отменено пользователем"
    exit 0
fi

echo ""
echo "🛑 Остановка всех контейнеров..."
docker stop $(docker ps -aq) 2>/dev/null || true

echo "🗑️  Удаление всех контейнеров..."
docker rm $(docker ps -aq) 2>/dev/null || true

echo "🖼️  Удаление всех образов..."
docker rmi $(docker images -q) -f 2>/dev/null || true

echo "💾 Удаление всех томов..."
docker volume rm $(docker volume ls -q) 2>/dev/null || true

echo "🌐 Удаление всех сетей..."
docker network rm $(docker network ls -q) 2>/dev/null || true

echo "🧹 Системная очистка..."
docker system prune -af --volumes

echo ""
echo "✅ Docker полностью очищен!"
echo "Теперь можно запустить: make quick-start"

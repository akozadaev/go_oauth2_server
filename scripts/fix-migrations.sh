#!/bin/bash

# Скрипт для исправления проблем с миграциями

set -e

echo "🔧 Исправление проблем с миграциями..."
echo ""

# Проверяем, что PostgreSQL запущен
if ! docker-compose ps postgres | grep -q "Up"; then
    echo "❌ PostgreSQL не запущен. Запустите docker-compose up -d postgres"
    exit 1
fi

echo "🗄️ Подключение к базе данных..."
echo ""

# Проверяем состояние миграций
echo "📊 Проверка состояния миграций:"
docker-compose exec postgres psql -U oauth2_user -d oauth2_db -c "
SELECT version, dirty, applied_at 
FROM schema_migrations 
ORDER BY version DESC;
" 2>/dev/null || echo "❌ Таблица миграций не найдена или недоступна"

echo ""
echo "🧹 Очистка грязного состояния миграций..."

# Очищаем грязное состояние миграций
docker-compose exec postgres psql -U oauth2_user -d oauth2_db -c "
-- Удаляем грязные записи миграций
DELETE FROM schema_migrations WHERE dirty = true;

-- Проверяем результат
SELECT version, dirty, applied_at 
FROM schema_migrations 
ORDER BY version DESC;
" 2>/dev/null || echo "❌ Не удалось очистить миграции"

echo ""
echo "✅ Исправление миграций завершено!"
echo ""
echo "🔄 Теперь можно перезапустить сервисы:"
echo "   docker-compose restart oauth2-server"
echo ""
echo "📚 Или полный перезапуск:"
echo "   docker-compose down && docker-compose up -d" 
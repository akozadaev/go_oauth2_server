#!/bin/bash

# Скрипт очистки токенов

set -e

echo "🧹 Очистка токенов OAuth2 сервера..."
echo ""

# Проверяем, что PostgreSQL запущен
if ! docker-compose ps postgres | grep -q "Up"; then
    echo "❌ PostgreSQL не запущен. Запустите docker-compose up -d postgres"
    exit 1
fi

echo "🗄️ Подключение к базе данных..."
docker-compose exec postgres psql -U oauth2_user -d oauth2_db -c "
-- Удаляем истекшие токены
DELETE FROM oauth2_tokens 
WHERE access_expires_at < NOW() 
   OR (refresh_expires_at IS NOT NULL AND refresh_expires_at < NOW());

-- Показываем статистику
SELECT 
    'Total tokens' as metric, 
    COUNT(*) as count 
FROM oauth2_tokens
UNION ALL
SELECT 
    'Expired tokens' as metric, 
    COUNT(*) as count 
FROM oauth2_tokens 
WHERE access_expires_at < NOW() 
   OR (refresh_expires_at IS NOT NULL AND refresh_expires_at < NOW())
UNION ALL
SELECT 
    'Valid tokens' as metric, 
    COUNT(*) as count 
FROM oauth2_tokens 
WHERE access_expires_at >= NOW() 
   AND (refresh_expires_at IS NULL OR refresh_expires_at >= NOW());
"

echo ""
echo "✅ Очистка завершена!"
echo ""
echo "📊 Статистика токенов обновлена"

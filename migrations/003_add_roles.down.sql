-- Удаление индексов
DROP INDEX IF EXISTS idx_users_username;
DROP INDEX IF EXISTS idx_clients_domain;
DROP INDEX IF EXISTS idx_clients_user_id;

-- Удаление тестовых данных
DELETE FROM users WHERE username IN ('admin', 'developer');

-- Удаление расширения
DROP EXTENSION IF EXISTS "uuid-ossp"; 
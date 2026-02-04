-- init.sql
-- Инициализация базы данных для OAuth2 сервера

-- Создаем пользователя (если не существует)
DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'oauth2_user') THEN
        CREATE USER oauth2_user WITH PASSWORD 'oauth2_password';
    END IF;
END
$$;

-- Создаем базу данных (если не существует)
SELECT 'CREATE DATABASE oauth2_db OWNER oauth2_user'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'oauth2_db')\gexec

-- Предоставляем права пользователю
GRANT ALL PRIVILEGES ON DATABASE oauth2_db TO oauth2_user;

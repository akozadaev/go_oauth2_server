-- Инициализация базы данных
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Создание дополнительных индексов для производительности
CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
CREATE INDEX IF NOT EXISTS idx_clients_domain ON clients(domain);
CREATE INDEX IF NOT EXISTS idx_clients_user_id ON clients(user_id);

-- Создание пользователя для мониторинга
CREATE USER monitoring WITH PASSWORD 'monitoring_password';
GRANT CONNECT ON DATABASE oauth2_db TO monitoring;
GRANT USAGE ON SCHEMA public TO monitoring;
GRANT SELECT ON ALL TABLES IN SCHEMA public TO monitoring;

-- Вставка тестовых данных для разработки
INSERT INTO users (id, username, password, created_at)
VALUES (
           uuid_generate_v4(),
           'admin',
           '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', -- password: admin
           NOW()
       ) ON CONFLICT (username) DO NOTHING;

INSERT INTO users (id, username, password, created_at)
VALUES (
           uuid_generate_v4(),
           'developer',
           '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', -- password: developer
           NOW()
       ) ON CONFLICT (username) DO NOTHING;

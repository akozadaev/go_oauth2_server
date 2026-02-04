-- Добавление индекса на username для оптимизации поиска пользователей
-- Это критично для производительности ValidateUser при высокой нагрузке
CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);


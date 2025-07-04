CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS users (
                                     id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    username VARCHAR(255) UNIQUE NOT NULL,
    password VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
    );

CREATE TABLE IF NOT EXISTS clients (
                                       id VARCHAR(255) PRIMARY KEY,
    secret VARCHAR(255) NOT NULL,
    domain VARCHAR(255) NOT NULL,
    user_id VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
    );

-- Insert a test user (password: P@$$w0rd)
-- Пароль сгенерирован здесь https://bcrypt-generator.com/
INSERT INTO users (id, username, password, created_at)
VALUES (
           uuid_generate_v4(),
           'testuser',
           '$2a$10$pyrArBhvQOu3W69lPsV8Vu4oGIoWlnBUMqMI9eNfT.LTh5HCbZdwe',
           NOW()
       ) ON CONFLICT (username) DO NOTHING;

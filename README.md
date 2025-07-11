# OAuth2 Server

[![Go Version](https://img.shields.io/badge/Go-1.23.4-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)]([LICENSE](LICENSE))
[![Docker](https://img.shields.io/badge/Docker-Ready-blue.svg)]([Dockerfile](Dockerfile))
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-16-blue.svg)](https://postgresql.org/)
[![Build Status](https://img.shields.io/badge/Build-Passing-brightgreen.svg)](#)

OAuth2 сервер на Go 1.23.4, на базе PostgreSQL

## Возможности

- Authorization Code Grant
- Client Credentials Grant
- Resource Owner Password Credentials Grant
- Refresh Token Grant
- Регистрация клиентов
- Авторизация пользователей
- JWT токены с настраиваемым временем жизни
- PostgreSQL база данных
- Автоматические миграции БД
- Структурированное логирование
- Health check
- CORS поддержка

## Быстрый старт с Docker

### 1. Клонируйте проект:
```bash
git clone https://github.com/akozadaev/go_oauth2_server.git
cd go_oauth2_server
```

### 2. Запустите с помощью Docker Compose:
```bash
# Быстрый запуск
./scripts/quick-start.sh

# Или вручную
docker-compose up -d
```

### 3. Проверьте работу:
```bash
# Статус контейнеров
docker-compose ps

# Health check
curl http://localhost:8080/health

# Логи
docker-compose logs
```

## Установка и запуск (локально)

1. Клонируйте проект:
```bash
git clone https://github.com/akozadaev/go_oauth2_server.git
cd go_oauth2_server
```

2. Убедитесь, что у вас установлена соответствующая версия Go:
```bash
go version
```

3. Установите зависимости:
```bash
go mod tidy
```

4. Настройте PostgreSQL и создайте базу данных:
```sql
CREATE DATABASE oauth2_db;
CREATE USER oauth2_user WITH PASSWORD 'your_password';
GRANT ALL PRIVILEGES ON DATABASE oauth2_db TO oauth2_user;
```

5. Настройте переменные окружения в `.env`:
```env
PORT=8080
DATABASE_URL=postgres://oauth2_user:your_password@localhost:5432/oauth2_db?sslmode=disable
JWT_SECRET=your-super-secret-jwt-key-change-this-in-production-make-it-at-least-32-characters-long
TOKEN_EXPIRATION_MINUTES=60
REFRESH_EXPIRATION_HOURS=168
LOG_LEVEL=info
```

6. Запустите сервер:
```bash
go run ./cmd/server/main.go
```

## Docker команды

### Основные команды:
```bash
# Запуск всех сервисов
docker-compose up -d

# Остановка сервисов
docker-compose down

# Перезапуск
docker-compose restart

# Просмотр логов
docker-compose logs

# Статус контейнеров
docker-compose ps

# Пересборка образов
docker-compose build --no-cache

# Принудительная пересборка и запуск
docker-compose up --force-recreate --build
```

### Диагностика:
```bash
# Полная диагностика
./scripts/diagnose.sh

# Устранение неполадок
./scripts/troubleshoot.sh

# Отладка контейнера
./scripts/container-debug.sh
```

### Разработка:
```bash
# Настройка среды разработки
./scripts/dev-setup.sh

# Запуск в режиме разработки
./scripts/dev-run.sh

# Очистка токенов
./scripts/cleanup-tokens.sh
```

### Решение проблем:
```bash
# Исправление миграций
./scripts/fix-migrations.sh

# Полная очистка и перезапуск
./scripts/clean-restart.sh

# Быстрый перезапуск
./scripts/restart-fixed.sh
```

## API Endpoints

### 1. Health Check
```bash
GET /health
```

### 2. Регистрация клиента
```bash
POST /clients
Content-Type: application/json

{
  "id": "client_id",
  "secret": "client_secret",
  "domain": "http://localhost:3000"
}
```

### 3. Регистрация пользователя
```bash
POST /users
Content-Type: application/json

{
  "username": "testuser",
  "password": "testpass"
}
```

### 4. Authorization Code Grant
```bash
# Шаг 1: Получение authorization code
GET /authorize?response_type=code&client_id=CLIENT_ID&redirect_uri=http://localhost:3000/callback&scope=read&state=random_state

# Шаг 2: Обмен code на токен
POST /token
Content-Type: application/x-www-form-urlencoded

grant_type=authorization_code&code=AUTHORIZATION_CODE&client_id=CLIENT_ID&client_secret=CLIENT_SECRET&redirect_uri=http://localhost:3000/callback
```

### 5. Client Credentials Grant
```bash
POST /token
Content-Type: application/x-www-form-urlencoded

grant_type=client_credentials&client_id=CLIENT_ID&client_secret=CLIENT_SECRET&scope=read
```

### 6. Resource Owner Password Credentials
```bash
POST /token
Content-Type: application/x-www-form-urlencoded

grant_type=password&username=testuser&password=testpass&client_id=CLIENT_ID&client_secret=CLIENT_SECRET
```

### 7. Refresh Token
```bash
POST /token
Content-Type: application/x-www-form-urlencoded

grant_type=refresh_token&refresh_token=REFRESH_TOKEN&client_id=CLIENT_ID&client_secret=CLIENT_SECRET
```

### 8. Token Introspection
```bash
POST /introspect
Content-Type: application/json

{
  "token": "ACCESS_TOKEN",
  "token_type_hint": "access_token"
}
```

## Структура проекта

```
oauth2-server/
├── cmd/server/main.go          # Точка входа
├── internal/
│   ├── config/config.go        # Конфигурация
│   ├── handlers/handlers.go    # HTTP хендлеры
│   ├── models/models.go        # Модели данных
│   └── storage/postgres.go     # Работа с БД
├── migrations/                 # Миграции БД
│   ├── 001_initial.up.sql
│   └── 001_initial.down.sql
├── scripts/                    # Скрипты для Docker
├── docker-compose.yml          # Docker Compose конфигурация
├── Dockerfile                  # Docker образ
├── .env                        # Переменные окружения
├── go.mod                      # Зависимости Go
└── README.md                   # Документация
```

## Устранение неполадок

### Проблемы с Docker:

1. **Контейнеры не запускаются:**
   ```bash
   # Остановите все контейнеры
   docker-compose down --remove-orphans
   
   # Пересоберите образы
   docker-compose build --no-cache
   
   # Запустите заново
   docker-compose up -d
   ```

2. **Проблемы с портами:**
   ```bash
   # Проверьте занятые порты
   lsof -i :8080
   lsof -i :5433
   
   # Остановите конфликтующие процессы
   ./scripts/troubleshoot.sh
   ```

3. **Проблемы с базой данных:**
   ```bash
   # Проверьте логи PostgreSQL
   docker-compose logs postgres
   
   # Перезапустите только БД
   docker-compose restart postgres
   ```

4. **Проблемы с приложением:**
   ```bash
   # Проверьте логи приложения
   docker-compose logs oauth2-server
   
   # Запустите диагностику
   ./scripts/diagnose.sh
   ```

### Проблемы с миграциями:

1. **Ошибка "Dirty database version":**
   ```bash
   # Исправьте состояние миграций
   ./scripts/fix-migrations.sh
   
   # Перезапустите приложение
   docker-compose restart oauth2-server
   ```

2. **Проблемы с пользователем PostgreSQL:**
   ```bash
   # Полная очистка и перезапуск
   ./scripts/clean-restart.sh
   ```

3. **Проблемы с инициализацией БД:**
   ```bash
   # Удалите volumes и перезапустите
   docker-compose down --volumes
   docker-compose up -d
   ```

### Проблемы с локальной разработкой:

1. **Go не найден:**
   ```bash
   # Установите Go 1.23+
   # https://golang.org/dl/
   ```

2. **Проблемы с зависимостями:**
   ```bash
   go mod tidy
   go mod download
   ```

3. **Проблемы с БД:**
   ```bash
   # Убедитесь что PostgreSQL запущен
   sudo systemctl status postgresql
   
   # Создайте БД и пользователя
   sudo -u postgres psql
   CREATE DATABASE oauth2_db;
   CREATE USER oauth2_user WITH PASSWORD 'your_password';
   GRANT ALL PRIVILEGES ON DATABASE oauth2_db TO oauth2_user;
   ```

## Логирование

Приложение использует структурированное логирование с `log/slog`:

```json
{
  "time": "2024-01-15T10:30:00Z",
  "level": "INFO",
  "msg": "Server starting",
  "port": "8080"
}
```

## Безопасность

- Используйте сильные JWT секреты (минимум 32 символа)
- Настройте HTTPS в продакшене
- Ограничьте доступ к базе данных
- Регулярно обновляйте зависимости
- Мониторьте логи на предмет подозрительной активности

## Лицензия

MIT License - см. [LICENSE](LICENSE) для деталей.

## Поддержка

Если у вас есть вопросы или проблемы:

1. Проверьте раздел [Устранение неполадок](#устранение-неполадок)
2. Запустите диагностику: `./scripts/diagnose.sh`
3. Создайте issue в GitHub
4. Обратитесь к документации в папке `docs/`
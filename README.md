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

## Установка и запуск

1. Клонируйте проект:
https://github.com/akozadaev/go_oauth2_server.git

2. Убедитесь, что у вас установлена соответствующая версия Go:
```bash
go version
```

3. Установите зависимости:
```bash
go mod tidy
```

34. Настройте PostgreSQL и создайте базу данных:
```sql
CREATE DATABASE oauth2_db;
CREATE USER oauth2_user WITH PASSWORD 'your_password';
GRANT ALL PRIVILEGES ON DATABASE oauth2_db TO oauth2_user;
```

4. Настройте переменные окружения в `.env`:
```env
PORT=8080
DATABASE_URL=postgres://oauth2_user:your_password@localhost:5432/oauth2_db?sslmode=disable
JWT_SECRET=your-super-secret-jwt-key-change-this-in-production-make-it-at-least-32-characters-long
TOKEN_EXPIRATION_MINUTES=60
REFRESH_EXPIRATION_HOURS=168
LOG_LEVEL=info
```

5. Запустите сервер для проверки работоспособности:
```bash
go run cmd/server/main.go
```

## Проект использует Makefile для автоматизации типичных задач разработки и сборки.
[Makefile](Makefile)

### Установка зависимостей и утилит
```bash
make tools
```

Устанавливает необходимые инструменты для линтинга:
- revive — альтернативный быстрый линтер.
- golangci-lint — объединяет множество линтеров Go.

### Генерация кода
```bash
make generate
```
Выполняет все директивы //go:generate в проекте.

### Сборка
### Локальная сборка
```bash
make build
```
Собирает бинарный файл сервера go_oauth2_server из ./cmd/server/.

### Сборка для продакшена (Linux AMD64)

```bash
make release
```

Создает оптимизированный бинарник и упаковывает его в go_oauth2_server.zip.

### Форматирование кода
```bash
make fmt
```
Форматирует весь Go-код с помощью gofmt -s -w.

### Тестирование
```bash
make test
```
Запускает все модульные тесты с подробным выводом.

### Линтинг
#### Полный линтинг
```bash
make lint-full
```
Запускает golangci-lint для всего проекта. При необходимости скачивает нужную версию.

#### Автоисправление проблем
```bash
make lint-fix
```
Автоматически исправляет найденные линтерами проблемы (если возможно).

### Финальная проверка перед коммитом
```bash
make check
```
Запускает последовательность команд:
- fmt — форматирование кода.
- lint-full — линтинг.
- test — тестирование.

## Требования
- Go 1.20+ (желательно 1.23.4)
- Linux/macOS/WSL (или любая Unix-подобная среда)
- make

## Структура сборки
- Исходный код сервера: ./cmd/server
- Итоговый бинарник: ./go_oauth2_server
- Архив сборки: ./go_oauth2_server.zip

## Использование в CI
[lint.yml](.github/workflows/lint.yml)
![img.png](docs/img.png)

## Лицензия
MIT — используй свободно.

## API Endpoints

### Health Check
```bash
curl http://localhost:8080/health
```

### 1. Регистрация клиента

```bash
curl -X POST http://localhost:8080/clients \
  -H "Content-Type: application/json" \
  -d '{
    "domain": "https://example.com",
    "user_id": "test-user-id",
    "redirect_uris": ["https://example.com/callback"],
    "grant_types": ["authorization_code", "client_credentials", "refresh_token"]
  }'
```

### 2. Authorization Code Flow

#### Шаг 1: Авторизация пользователя
```bash
curl -X POST http://localhost:8080/authorize \
  -H "Content-Type: application/json" \
  -d '{
    "response_type": "code",
    "client_id": "your-client-id",
    "redirect_uri": "https://example.com/callback",
    "scope": "read write",
    "state": "random-state-string",
    "username": "testuser",
    "password": "$2a$10$pyrArBhvQOu3W69lPsV8Vu4oGIoWlnBUMqMI9eNfT.LTh5HCbZdwe"
  }'
```

#### Шаг 2: Обмен кода на токен
```bash
curl -X POST http://localhost:8080/token \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=authorization_code&code=AUTH_CODE&redirect_uri=https://example.com/callback&client_id=CLIENT_ID&client_secret=CLIENT_SECRET"
```

### 3. Client Credentials Flow

```bash
curl -X POST http://localhost:8080/token \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=client_credentials&client_id=CLIENT_ID&client_secret=CLIENT_SECRET&scope=read"
```

### 4. Resource Owner Password Credentials

```bash
curl -X POST http://localhost:8080/token \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=password&username=testuser&password=testpass&client_id=CLIENT_ID&client_secret=CLIENT_SECRET"
```

### 5. Refresh Token

```bash
curl -X POST http://localhost:8080/token \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=refresh_token&refresh_token=REFRESH_TOKEN&client_id=CLIENT_ID&client_secret=CLIENT_SECRET"
```

### 6. Token Introspection

```bash
curl -X POST http://localhost:8080/introspect \
  -H "Content-Type: application/json" \
  -d '{
    "token": "ACCESS_TOKEN",
    "token_type_hint": "access_token"
  }'
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
├── .env                        # Переменные окружения
├── go.mod                      # Зависимости Go
└── README.md                   # Документация
```

## Логирование

Приложение использует структурированное логирование с `log/slog`:

```json
{
  "time": "2025-01-15T10:30:45.123Z",
  "level": "INFO",
  "msg": "Request processed",
  "method": "POST",
  "path": "/token",
  "status": 200,
  "duration": "15.2ms",
  "ip": "127.0.0.1:54321"
}
```

## Мониторинг

Health check endpoint предоставляет информацию о состоянии сервиса:

```bash
curl http://localhost:8080/health
```

Ответ:
```json
{
  "status": "healthy",
  "timestamp": 1705312245,
  "version": "1.0.0"
}
```

## Пример использования с реальными данными:
0. Регистрация пользователя
```bash
curl --location 'http://localhost:8080/users' \
--header 'Content-Type: application/json' \
--data '{
    "username": "testuser1",
    "password": "testpass"
}'
```
Ответ:
```json
{
    "created_at": 1751959089,
    "user_id": "225e775b-8107-4753-8636-e6aef80350b5",
    "username": "testuser1"
}
```
1. Регистрация клиента:

```bash
curl --location 'http://localhost:8080/clients' \
--header 'Content-Type: application/json' \
--data '{
    "domain": "https://example.com",
    "user_id": "225e775b-8107-4753-8636-e6aef80350b5",
    "redirect_uris": ["https://example.com/callback"],
    "grant_types": ["authorization_code", "client_credentials", "password", "refresh_token"]
  }'
```

Ответ:
```json
{
   "client_id": "8fb993b4-75e1-4146-830f-ed1dfe6cf0a4",
   "client_secret": "7c1dfcdc-88cc-491f-85ae-14f349108b3a",
   "created_at": 1751959195,
   "domain": "https://example.com",
   "user_id": "225e775b-8107-4753-8636-e6aef80350b5"
}
```
2. Получение токена с использованием полученных client_id и client_secret:

    2.1. Client Credentials Grant:
```bash
curl --location 'http://localhost:8080/token' \
--header 'Content-Type: application/x-www-form-urlencoded' \
--data-urlencode 'grant_type=client_credentials' \
--data-urlencode 'client_id=e7fb9eea-11e8-4ac7-9789-8e5903ba9472' \
--data-urlencode 'client_secret=31feb658-e1df-4411-bf2c-cb0c91cba1d1' \
--data-urlencode 'scope=read'
```

Ответ:
```json
{
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhdWQiOiJlN2ZiOWVlYS0xMWU4LTRhYzctOTc4OS04ZTU5MDNiYTk0NzIiLCJleHAiOjE3NTE5NjMxMzIsImlhdCI6MTc1MTk1NTkzMiwic3ViIjoiIn0.8sck8g3CI-1Z4XjRxrHncEwdAChvSXE-sSaNyKfdaoo",
    "expires_in": 7200,
    "scope": "read",
    "token_type": "Bearer"
}
```

   2.2. Password Grant:

```bash
curl --location 'http://localhost:8080/token' \
--header 'Content-Type: application/x-www-form-urlencoded' \
--data-urlencode 'grant_type=password' \
--data-urlencode 'username=testuser' \
--data-urlencode 'password=P@$$w0rd' \
--data-urlencode 'client_id=e7fb9eea-11e8-4ac7-9789-8e5903ba9472' \
--data-urlencode 'client_secret=31feb658-e1df-4411-bf2c-cb0c91cba1d1' \
--data-urlencode 'scope=read write'
```

Ответ:

```json
{
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhdWQiOiJlN2ZiOWVlYS0xMWU4LTRhYzctOTc4OS04ZTU5MDNiYTk0NzIiLCJleHAiOjE3NTE5NjMyODcsImlhdCI6MTc1MTk1NjA4Nywic3ViIjoiYjRmYTZhYzEtOWY0OC00ZjMzLWJlN2MtZmI5OGEzY2Q4NGUyIn0.Jcr00EP7a6gqjnDEQ2A5Bi1_8tvW21zBJUXjaK2hxvU",
    "expires_in": 7200,
    "refresh_token": "hBiyP8ApJd7kd-YTe4LFRhgu-1eLsGa0-UwyzU9Ug58=",
    "scope": "read write",
    "token_type": "Bearer"
}
```

3. Проверка полученного токена:
```bash
curl --location 'http://localhost:8080/introspect' \
--header 'Content-Type: application/json' \
--data '{"token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhdWQiOiJlN2ZiOWVlYS0xMWU4LTRhYzctOTc4OS04ZTU5MDNiYTk0NzIiLCJleHAiOjE3NTE5NjMyODcsImlhdCI6MTc1MTk1NjA4Nywic3ViIjoiYjRmYTZhYzEtOWY0OC00ZjMzLWJlN2MtZmI5OGEzY2Q4NGUyIn0.Jcr00EP7a6gqjnDEQ2A5Bi1_8tvW21zBJUXjaK2hxvU"}'
```

Ответ:
```json
{
    "active": true,
    "client_id": "e7fb9eea-11e8-4ac7-9789-8e5903ba9472",
    "username": "b4fa6ac1-9f48-4f33-be7c-fb98a3cd84e2",
    "exp": 1751963287
}
```
4. Access Token истек, используем Refresh Token:
```bash
curl -X POST http://localhost:8080/token \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=refresh_token&refresh_token=hBiyP8ApJd7kd-YTe4LFRhgu-1eLsGa0-UwyzU9Ug58=&client_id=e7fb9eea-11e8-4ac7-9789-8e5903ba9472&client_secret=31feb658-e1df-4411-bf2c-cb0c91cba1d1"
```

Ответ:
```bash
{
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhdWQiOiJlN2ZiOWVlYS0xMWU4LTRhYzctOTc4OS04ZTU5MDNiYTk0NzIiLCJleHAiOjE3NTE5NjQ2MjEsImlhdCI6MTc1MTk1NzQyMSwic3ViIjoiYjRmYTZhYzEtOWY0OC00ZjMzLWJlN2MtZmI5OGEzY2Q4NGUyIn0.gnR3BG18xGX7RACUkB-fkIwKtutrJmH8j8EzBVlR5yk",
    "expires_in": 7200,
    "refresh_token": "uLtHiaM7ATYhmmAuLIcI9hMeVKLMuR5_neZY3HjC43U=",
    "scope": "read write",
    "token_type": "Bearer"
}
```

## Пример работы логера при обработке запросов:

```json
{"time":"2025-07-08T09:02:15.630008026+03:00","level":"INFO","msg":"Server starting","port":"8080"}
```
```json
{"time":"2025-07-08T09:02:19.117785708+03:00","level":"INFO","msg":"Client registered successfully","client_id":"e7fb9eea-11e8-4ac7-9789-8e5903ba9472","domain":"https://example.com"}
```
```json
{"time":"2025-07-08T09:02:19.117849627+03:00","level":"INFO","msg":"Request processed","method":"POST","path":"/clients","status":201,"duration":2847607,"ip":"[::1]:38550","user_agent":"PostmanRuntime/7.44.1"}
```
```json
{"time":"2025-07-08T09:25:32.949975094+03:00","level":"INFO","msg":"Request processed","method":"POST","path":"/token","status":200,"duration":206633,"ip":"[::1]:60598","user_agent":"PostmanRuntime/7.44.1"}
```
```json
{"time":"2025-07-08T09:28:07.6748477+03:00","level":"INFO","msg":"Request processed","method":"POST","path":"/token","status":200,"duration":125439360,"ip":"[::1]:48294","user_agent":"PostmanRuntime/7.44.1"}
```
```json
{"time":"2025-07-08T09:30:33.159097703+03:00","level":"INFO","msg":"Client registered successfully","client_id":"b9ef30de-5570-4318-a5f7-255f51919ddd","domain":"https://example.com"}
```
```json
{"time":"2025-07-08T09:30:33.159133086+03:00","level":"INFO","msg":"Request processed","method":"POST","path":"/clients","status":201,"duration":7966379,"ip":"[::1]:50266","user_agent":"PostmanRuntime/7.44.1"}
```
```json
{"time":"2025-07-08T09:32:46.216333817+03:00","level":"INFO","msg":"Request processed","method":"POST","path":"/authorize","status":302,"duration":76835896,"ip":"[::1]:39278","user_agent":"PostmanRuntime/7.44.1"}
```
```json
{"time":"2025-07-08T09:33:01.465040133+03:00","level":"INFO","msg":"Client registered successfully","client_id":"82797c7f-e477-4d8a-b720-6d085fbe7895","domain":"https://example.com"}
```
```json
{"time":"2025-07-08T09:33:01.465086208+03:00","level":"INFO","msg":"Request processed","method":"POST","path":"/clients","status":201,"duration":8297536,"ip":"[::1]:39278","user_agent":"PostmanRuntime/7.44.1"}
```
```json
{"time":"2025-07-08T09:34:07.751195263+03:00","level":"INFO","msg":"Request processed","method":"POST","path":"/introspect","status":200,"duration":104479,"ip":"[::1]:39278","user_agent":"PostmanRuntime/7.44.1"}
```
```json
{"time":"2025-07-08T09:38:54.825218219+03:00","level":"INFO","msg":"Request processed","method":"POST","path":"/introspect","status":200,"duration":99970,"ip":"[::1]:59552","user_agent":"PostmanRuntime/7.44.1"}
```
```json
{"time":"2025-07-08T09:40:30.556867094+03:00","level":"INFO","msg":"Request processed","method":"POST","path":"/introspect","status":200,"duration":106009,"ip":"[::1]:46988","user_agent":"PostmanRuntime/7.44.1"}
```
```json
{"time":"2025-07-08T09:41:21.621930672+03:00","level":"INFO","msg":"Request processed","method":"POST","path":"/token","status":401,"duration":61084,"ip":"[::1]:46988","user_agent":"PostmanRuntime/7.44.1"}
```
```json
{"time":"2025-07-08T09:50:21.922219294+03:00","level":"INFO","msg":"Request processed","method":"POST","path":"/token","status":200,"duration":131809,"ip":"[::1]:57390","user_agent":"PostmanRuntime/7.44.1"}
```
## Проверка работоспособности, вспомогательные инструменты администирования

1. Установите прав на выполнение для скриптов
chmod +x scripts/*.sh
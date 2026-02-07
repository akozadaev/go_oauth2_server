# Тестирование OAuth2 Server

Этот проект использует Docker для интеграционного тестирования с реальными PostgreSQL контейнерами.

## 🎯 Результаты тестирования

✅ **Успешно работают:**
- Unit тесты для storage (PostgresStore)
- Unit тесты для handlers
- Интеграционные тесты для регистрации пользователей и клиентов
- Health check тесты

⚠️ **Требует доработки:**
- Полный OAuth2 flow (авторизация -> токен -> интроспекция) - есть проблема с получением клиента

## Структура тестов

```
tests/
├── integration_test.go    # Интеграционные тесты
└── README.md             # Этот файл

internal/
├── storage/
│   └── postgres_test.go  # Тесты для storage
└── handlers/
    └── handlers_test.go  # Тесты для handlers

scripts/
└── test-with-docker.sh   # Скрипт для запуска тестов с Docker
```

## Запуск тестов

### Все тесты (требует локальную БД)
```bash
make test
```

### Только unit тесты
```bash
make test-unit
```

### Только интеграционные тесты
```bash
make test-integration
```

### Тесты с покрытием
```bash
make test-coverage
```

### Тесты с Docker (рекомендуется)
```bash
make test-docker
```

### Тесты с Docker Compose
```bash
make test-docker-compose
```

## Требования

- Go 1.23+
- Docker
- Docker Compose (опционально)

## Что тестируется

### ✅ Unit тесты
- Создание и получение пользователей
- Создание и получение клиентов
- Валидация пользователей и клиентов
- Работа с токенами
- Health check

### ✅ Интеграционные тесты
- Регистрация пользователей и клиентов
- Проверка health endpoint

### ⚠️ Требует доработки
- Полный OAuth2 flow (авторизация -> токен -> интроспекция)

## Используемые технологии

- **Docker**: Для запуска PostgreSQL в контейнерах
- **testify**: Для assertions и require
- **httptest**: Для тестирования HTTP handlers
- **chi**: Для роутинга в тестах

## Настройка тестов

### Автоматическая настройка (рекомендуется)

Тесты автоматически:
1. Запускают PostgreSQL контейнер
2. Создают тестовую базу данных
3. Применяют миграции
4. Выполняют тесты
5. Очищают ресурсы

```bash
make test-docker
```

### Ручная настройка

Если вы хотите запустить тесты вручную:

1. Запустите PostgreSQL контейнер:
```bash
docker run -d \
    --name test-postgres \
    -e POSTGRES_DB=test_db \
    -e POSTGRES_USER=test_user \
    -e POSTGRES_PASSWORD=test_password \
    -p 5433:5432 \
    postgres:15-alpine
```

2. Установите переменную окружения:
```bash
export TEST_DATABASE_URL="postgres://test_user:test_password@localhost:5433/test_db?sslmode=disable"
```

3. Запустите тесты:
```bash
go test -v ./...
```

4. Очистите ресурсы:
```bash
docker stop test-postgres
docker rm test-postgres
```

## Переменные окружения

- `TEST_DATABASE_URL`: URL для подключения к тестовой базе данных
  - По умолчанию: `postgres://test_user:test_password@localhost:5432/test_db?sslmode=disable`

## Добавление новых тестов

1. Создайте новый тестовый файл с суффиксом `_test.go`
2. Используйте `getTestDB()` для получения подключения к БД
3. Используйте `cleanupTestDB()` для очистки данных
4. Не забудьте добавить `defer` для очистки ресурсов
5. Используйте уникальные имена для тестовых данных

Пример:
```go
func TestNewFeature(t *testing.T) {
    db := getTestDB(t)
    defer db.Close()
    defer cleanupTestDB(db)
    
    // Ваши тесты здесь
}
```

## Troubleshooting

### Ошибка подключения к базе данных
Убедитесь, что:
1. Docker запущен
2. PostgreSQL контейнер запущен
3. Порт 5433 свободен
4. Переменная `TEST_DATABASE_URL` установлена правильно

### Ошибка "password authentication failed"
Проверьте, что:
1. PostgreSQL контейнер запущен с правильными переменными окружения
2. База данных `test_db` создана
3. Пользователь `test_user` существует с правильным паролем

### Ошибка "duplicate key value violates unique constraint"
Это означает, что данные не очищаются между тестами. Убедитесь, что:
1. Используются уникальные имена для тестовых данных
2. Функция `cleanupTestDB()` вызывается перед каждым тестом
3. Тесты не конфликтуют друг с другом

### Медленные тесты
Тесты могут быть медленными из-за:
1. Ожидания готовности PostgreSQL
2. Создания и удаления таблиц
3. Очистки данных между тестами

Для ускорения можно:
1. Использовать in-memory базу данных для unit тестов
2. Оптимизировать запросы
3. Использовать параллельное выполнение тестов

## Статус тестирования

| Тест | Статус | Описание |
|------|--------|----------|
| TestHandler_Health | ✅ PASS | Проверка health endpoint |
| TestHandler_RegisterUser | ✅ PASS | Регистрация пользователя |
| TestHandler_RegisterClient | ✅ PASS | Регистрация клиента |
| TestHandler_Integration | ✅ PASS | Интеграционный тест handlers |
| TestPostgresStore_CreateClient | ✅ PASS | Создание клиента в БД |
| TestPostgresStore_CreateUser | ✅ PASS | Создание пользователя в БД |
| TestPostgresStore_ValidateUser | ✅ PASS | Валидация пользователя |
| TestPostgresStore_ValidateClient | ✅ PASS | Валидация клиента |
| TestClientStore_GetByID | ✅ PASS | Получение клиента по ID |
| TestPostgresStore_Integration | ✅ PASS | Интеграционный тест storage |
| TestOAuth2Flow_Integration | ⚠️ FAIL | Полный OAuth2 flow |
| TestHealthCheck_Integration | ✅ PASS | Health check интеграционный |
| TestUserRegistration_Integration | ✅ PASS | Регистрация пользователя интеграционный |
| TestClientRegistration_Integration | ✅ PASS | Регистрация клиента интеграционный | 
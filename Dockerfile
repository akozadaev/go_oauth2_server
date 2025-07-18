# Multi-stage build
FROM golang:1.23.4-alpine AS builder

# Устанавливаем необходимые пакеты
RUN apk add --no-cache git ca-certificates tzdata

# Создаем пользователя без root-доступа
RUN adduser -D -g '' appuser

# Рабочая директория
WORKDIR /build

# Копируем go.mod и go.sum для кеширования зависимостей
COPY go.mod go.sum ./

# Загружаем зависимости
RUN go mod download && go mod verify

# Копируем исходный код
COPY . .

# Очищаем vendor если есть и пересоздаем
RUN rm -rf vendor && go mod tidy

# Собираем приложение с -mod=readonly для избежания проблем с vendorприложения
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -mod=readonly \
    -ldflags='-w -s' \
    -o oauth2-server ./cmd/server/main.go

# Финальный образ
FROM alpine:latest

# Устанавливаем wget для health check
RUN apk --no-cache add ca-certificates wget

# Добавляем непривилегированного пользователя
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group

# Рабочая директория
WORKDIR /app

# Копируем бинарник и миграции
COPY --from=builder /build/oauth2-server .
COPY --from=builder /build/migrations ./migrations

# Права на исполняемый файл и директории
RUN chmod +x oauth2-server && \
    chown -R appuser:appuser /app

# Устанавливаем пользователя
USER appuser

# Открываем порт
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=60s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Запускаем приложение (с миграциями внутри)
CMD ["./oauth2-server"]

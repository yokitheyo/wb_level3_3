# Используем официальный Go образ как базовый для сборки
FROM golang:1.23.5-alpine AS builder

# Устанавливаем рабочую директорию
WORKDIR /app

# Копируем go mod и sum файлы
COPY go.mod go.sum ./

# Загружаем зависимости
RUN go mod download

# Копируем исходный код
COPY . .

# Собираем приложение
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/server

# Используем легковесный образ для запуска
FROM alpine:latest

# Устанавливаем ca-certificates для HTTPS запросов
RUN apk --no-cache add ca-certificates tzdata

# Создаем пользователя для безопасности
RUN adduser -D -s /bin/sh appuser

WORKDIR /app

# Копируем бинарный файл из builder стейджа
COPY --from=builder /app/main .

# Копируем конфигурационный файл
COPY --from=builder /app/config.yaml .

# Копируем миграции
COPY --from=builder /app/migrations ./migrations

# Меняем владельца файлов на appuser
RUN chown -R appuser:appuser /app

# Переключаемся на непривилегированного пользователя
USER appuser

# Открываем порт
EXPOSE 8080

# Команда запуска
CMD ["./main"]
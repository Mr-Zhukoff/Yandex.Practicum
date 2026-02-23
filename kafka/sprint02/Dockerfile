# Dockerfile для сборки Go приложений
FROM golang:1.21-alpine AS builder

# Установка необходимых инструментов
RUN apk add --no-cache git

# Установка рабочей директории
WORKDIR /app

# Копирование go.mod и go.sum для кэширования зависимостей
COPY go.mod go.sum ./
RUN go mod download

# Копирование исходного кода
COPY . .

# Сборка всех приложений
RUN go build -o /app/bin/processor cmd/processor/main.go
RUN go build -o /app/bin/producer cmd/producer/main.go
RUN go build -o /app/bin/consumer cmd/consumer/main.go

# Финальный образ
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Копирование бинарных файлов из builder
COPY --from=builder /app/bin/processor .
COPY --from=builder /app/bin/producer .
COPY --from=builder /app/bin/consumer .

# По умолчанию запускаем processor
CMD ["./processor"]

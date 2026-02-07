# Kafka sprint 1

## Архитектура приложения

Go-приложение (`app/`) состоит из:

- **Producer** — асинхронно отправляет сообщения в Kafka-топик каждые 500 мс.
- **SingleMessageConsumer** — считывает по одному сообщению, обрабатывает его и коммитит оффсет автоматически (`enable.auto.commit: true`). Группа: `single-message-consumer-group`.
- **BatchMessageConsumer** — считывает минимум 10 сообщений за один poll-цикл, обрабатывает их в цикле и один раз коммитит оффсет после обработки пачки (`consumer.Commit()`). Настройки: `fetch.min.bytes: 1024`, `fetch.max.wait.ms: 3000`. Группа: `batch-message-consumer-group`.

Консьюмеры работают параллельно (каждый в своей горутине) и считывают одни и те же сообщения благодаря уникальным `group.id`.

Приложение запускается в **2 экземплярах** через параметр `deploy.replicas: 2` в docker-compose.

При возникновении ошибок консьюмеры записывают сообщение в логи и продолжают работать.

## Запуск

Запускаем Docker Compose, чтобы создать и запустить контейнеры:

```bash
docker-compose up -d
```

Проверяем успешность запуска. Убедитесь, что контейнеры работают:

```bash
docker ps
```

Вы должны увидеть работающие контейнеры:
 - 1 для Zookeeper 
 - 3 для Kafka
 - 1 для Kafka-UI
 - 2 для kafka-app (реплики)

## Создание топика

Для создания топика с 3 партициями и 2 репликами выполните следующую команду:

```bash
docker exec sprint01-kafka-1-1 kafka-topics --create \
  --topic my-topic \
  --partitions 3 \
  --replication-factor 2 \
  --bootstrap-server kafka-1:29092
```

Проверить можно командой describe:

```bash
docker exec sprint01-kafka-1-1 kafka-topics --describe --topic my-topic --bootstrap-server kafka-1:29092
```

## Просмотр логов приложения

```bash
docker-compose logs -f kafka-app
```

## Структура проекта

```
├── docker-compose.yml
├── README.md
└── app/
    ├── Dockerfile
    ├── go.mod
    ├── main.go
    ├── producer/
    │   └── producer.go
    └── consumer/
        ├── single.go
        └── batch.go
```
Созданные файлы:
app/main.go — точка входа: создаёт producer и 2 consumer, запускает каждый в своей горутине, обрабатывает graceful shutdown через context.WithCancel + os.Signal.

app/producer/producer.go — асинхронный продюсер через kafka.Writer с Async: true и callback Completion для обработки delivery reports.

app/consumer/single.go — SingleMessageConsumer: читает по одному сообщению через ReadMessage() (авто-коммит через CommitInterval: time.Second). group.id = single-message-consumer-group.

app/consumer/batch.go — BatchMessageConsumer: собирает минимум 10 сообщений через FetchMessage() (без авто-коммита), обрабатывает пачку в цикле, затем один раз коммитит через CommitMessages(). Настройки: MinBytes: 1024 (fetch.min.bytes), MaxWait: 3s (fetch.max.wait.ms). group.id = batch-message-consumer-group.

app/Dockerfile — multi-stage build: golang:1.21 → debian:bookworm-slim.

docker-compose.yml — сервис kafka-app с deploy.replicas: 2 для запуска двух экземпляров.

Библиотека: segmentio/kafka-go v0.4.47 — чистый Go без CGO-зависимостей.
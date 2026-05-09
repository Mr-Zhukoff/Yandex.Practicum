# Задание 1. Развёртывание и настройка Kafka-кластера в Docker Desktop

## 1) Что развернуто

- Kafka-кластер из 3 брокеров: [`kafka-1`](docker-compose.yml), [`kafka-2`](docker-compose.yml), [`kafka-3`](docker-compose.yml)
- [`ZooKeeper`](docker-compose.yml)
- [`Schema Registry`](docker-compose.yml)

Конфигурация описана в [`docker-compose.yml`](docker-compose.yml).

## 2) Аппаратные ресурсы (рекомендация для Docker Desktop)

Для локального production-like стенда:

- CPU: **6 vCPU** (минимум 4)
- RAM: **10–12 GB** (минимум 8 GB)
- Disk (Docker): **не менее 40 GB**

Ориентир по брокерам:

- на каждый брокер: ~1 vCPU, 1.5–2.5 GB RAM, отдельный volume
- JVM heap брокера: `-Xms512m -Xmx512m` (см. [`KAFKA_HEAP_OPTS`](docker-compose.yml))

## 3) Параметры кластера и хранения

Ключевые параметры (заданы в [`docker-compose.yml`](docker-compose.yml)):

- `KAFKA_DEFAULT_REPLICATION_FACTOR=3`
- `KAFKA_MIN_INSYNC_REPLICAS=2`
- `KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR=3`
- `KAFKA_TRANSACTION_STATE_LOG_REPLICATION_FACTOR=3`
- `KAFKA_TRANSACTION_STATE_LOG_MIN_ISR=2`
- `KAFKA_LOG_CLEANUP_POLICY=delete`
- `KAFKA_LOG_RETENTION_MS=604800000` (7 суток)
- `KAFKA_LOG_SEGMENT_BYTES=134217728` (128 MB)

## 4) Топик, репликация и хранение

Создан топик `user-events`:

- partitions: 3
- replication factor: 3
- cleanup.policy: delete
- retention.ms: 604800000
- segment.bytes: 134217728

Команда:

```bash
docker exec kafka-1 kafka-topics --create --topic user-events --partitions 3 --replication-factor 3 --config cleanup.policy=delete --config retention.ms=604800000 --config segment.bytes=134217728 --bootstrap-server kafka-1:19092
```

Вывод [`kafka-topics --describe`](README.md):

```text
Topic: user-events	TopicId: QffduKNsSsaQ3MjT2JLmRQ	PartitionCount: 3	ReplicationFactor: 3	Configs: min.insync.replicas=2,cleanup.policy=delete,segment.bytes=134217728,retention.ms=604800000
	Topic: user-events	Partition: 0	Leader: 1	Replicas: 1,3,2	Isr: 1,3,2
	Topic: user-events	Partition: 1	Leader: 2	Replicas: 2,1,3	Isr: 2,1,3
	Topic: user-events	Partition: 2	Leader: 3	Replicas: 3,2,1	Isr: 3,2,1
```

## 5) Schema Registry и схема

Файл схемы:

- [`schemas/event.avsc`](schemas/event.avsc)

Payload для регистрации:

- [`schemas/register-user-events-value.json`](schemas/register-user-events-value.json)

Регистрация схемы:

```bash
curl.exe -s -X POST -H "Content-Type: application/vnd.schemaregistry.v1+json" --data-binary @schemas/register-user-events-value.json http://localhost:8081/subjects/user-events-value/versions
```

Ответ:

```json
{"id":1}
```

### Требуемые curl-ответы

`curl http://localhost:8081/subjects`

```json
["user-events-value"]
```

`curl -X GET http://localhost:8081/subjects/<название_схемы>/versions`

Для `user-events-value`:

```json
[1]
```

## 6) Продюсер/консьюмер (Go, JSON)

Код:

- Продюсер: [`main()`](app/cmd/producer/main.go:21)
- Консьюмер: [`main()`](app/cmd/consumer/main.go:12)
- Модуль: [`app/go.mod`](app/go.mod)

Запуск:

```bash
cd app
go mod tidy
go run ./cmd/producer/main.go
```

Для фиксации чтения сообщений использован консольный consumer Kafka в контейнере:

```bash
docker exec kafka-1 kafka-console-consumer --bootstrap-server kafka-1:19092 --topic user-events --from-beginning --max-messages 3
```

## 7) Логи успешной передачи

Логи продюсера (файл [`producer.log`](producer.log)):

```text
producer: sent event_id=evt-1 key=u-100 value={"event_id":"evt-1","event_type":"login","user_id":"u-100","event_ts":1778008890125,"source":"web"}
producer: sent event_id=evt-2 key=u-101 value={"event_id":"evt-2","event_type":"view_product","user_id":"u-101","event_ts":1778008890125,"source":"mobile"}
producer: sent event_id=evt-3 key=u-102 value={"event_id":"evt-3","event_type":"checkout","user_id":"u-102","event_ts":1778008890125,"source":"web"}
```

Логи консьюмера (файл [`consumer.log`](consumer.log)):

```text
{"event_id":"evt-1","event_type":"login","user_id":"u-100","event_ts":1778008890125,"source":"web"}
{"event_id":"evt-2","event_type":"view_product","user_id":"u-101","event_ts":1778008890125,"source":"mobile"}
{"event_id":"evt-3","event_type":"checkout","user_id":"u-102","event_ts":1778008890125,"source":"web"}
Processed a total of 3 messages
```

## 8) Состав артефактов

- Краткое описание шагов: [`README.md`](README.md)
- Информация по ресурсам: раздел 2 в [`README.md`](README.md)
- Скрипты конфигурации: [`docker-compose.yml`](docker-compose.yml), команды в [`README.md`](README.md)
- Параметры кластера: раздел 3 в [`README.md`](README.md)
- Ответы `curl`: раздел 5 в [`README.md`](README.md)
- Файл схемы: [`schemas/event.avsc`](schemas/event.avsc)
- Вывод `kafka-topics --describe`: раздел 4 в [`README.md`](README.md)
- Код продюсера/консьюмера: [`app/cmd/producer/main.go`](app/cmd/producer/main.go), [`app/cmd/consumer/main.go`](app/cmd/consumer/main.go)
- Логи передачи: [`producer.log`](producer.log), [`consumer.log`](consumer.log)

# Kafka SSL + ACL (3 брокера)

Проект поднимает кластер Kafka из 3 брокеров с SSL/mTLS, настраивает ACL, а также запускает `producer` и `consumer` в Docker.

## 1) Сгенерировать сертификаты

Запусти:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\generate-certs.ps1
```

Скрипт создаёт все нужные артефакты в каталоге `./certs`:
- keystore/truststore для брокеров
- keystore/truststore для клиентов (`admin`, `producer`, `consumer`)
- PEM-сертификаты и ключи для Go-клиентов
- CA-сертификат

## 2) Собрать и запустить сервисы

```powershell
docker compose up -d --build
```

Поднимутся:
- `zookeeper`
- `kafka-1`, `kafka-2`, `kafka-3`
- `kafka-ui` (http://localhost:8080)
- `producer`, `consumer` (внутри Docker-сети)

## 3) Создать топики и ACL

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\init-kafka.ps1
```

Скрипт:
- создаёт `topic-1` и `topic-2`
- выдаёт `producer` права `Write + Describe` на `topic-1` и `topic-2`
- выдаёт `consumer` права `Read + Describe` только на `topic-1`
- выдаёт `consumer` право `Read` на группу `consumer-group-1`

## 4) Проверить работу producer/consumer

Запуск вручную (рекомендуется для проверки логов):

```powershell
docker compose run --rm producer
docker compose run --rm consumer
```

Проверка логов:

```powershell
docker compose logs --no-log-prefix producer
docker compose logs --no-log-prefix consumer
```

Ожидаемое поведение:
- `producer` успешно пишет сообщения в `topic-1` и `topic-2`
- `consumer` читает сообщения из `topic-1`
- для `topic-2` у `consumer` нет прав чтения (ожидается ACL-ошибка или подтверждение отсутствия доступа)

## 5) Полезные команды

Остановить и удалить контейнеры:

```powershell
docker compose down
```

Пересоздать сертификаты и перезапустить всё заново:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\generate-certs.ps1
docker compose down
docker compose up -d --build
powershell -ExecutionPolicy Bypass -File .\scripts\init-kafka.ps1
```


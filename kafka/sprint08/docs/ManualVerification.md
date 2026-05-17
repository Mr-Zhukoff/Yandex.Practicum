# Инструкция по ручной проверке текущей реализации

Проверка рассчитана на текущий этап проекта:

- 2 Kafka-кластера по 3 брокера;
- TLS/mTLS;
- ACL;
- MirrorMaker 2;
- SHOP API;
- Goka-фильтр;
- PostgreSQL sink;
- CLIENT API;
- PostgreSQL search.

---

## 1. Предварительные требования

Нужно, чтобы были доступны:

```bash
docker
docker compose
go
openssl
keytool
```

Проверить:

```bash
docker --version
docker compose version
go version
openssl version
keytool -help
```

---

## 2. Сгенерировать сертификаты

Из директории `kafka/sprint08`:

```bash
./scripts/generate-certs.sh
```

После выполнения должна появиться директория:

```text
configs/kafka/certs/
```

В ней должны быть файлы вроде:

```text
ca.crt
kafka1.keystore.jks
kafka2.keystore.jks
kafka3.keystore.jks
kafka2-1.keystore.jks
kafka2-2.keystore.jks
kafka2-3.keystore.jks
kafka.truststore.jks
admin-ssl.properties
shop-api.crt
shop-api.key
client-api.crt
client-api.key
product-filter.crt
product-filter.key
postgres-sink.crt
postgres-sink.key
mirror-maker.keystore.jks
```

---

## 3. Запустить инфраструктуру

```bash
docker compose up -d kafka1 kafka2 kafka3 kafka2-1 kafka2-2 kafka2-3 postgres kafka-init kafka2-init kafka-acls kafka2-acls mirror-maker
```

Проверить статус:

```bash
docker compose ps
```

Ожидаемо:

- `kafka1`, `kafka2`, `kafka3` — running / healthy;
- `kafka2-1`, `kafka2-2`, `kafka2-3` — running / healthy;
- `postgres` — running / healthy;
- `kafka-init`, `kafka2-init` — exited successfully;
- `kafka-acls`, `kafka2-acls` — exited successfully;
- `mirror-maker` — running.

Если что-то упало:

```bash
docker compose logs <service-name>
```

Например:

```bash
docker compose logs kafka1
docker compose logs kafka-init
docker compose logs kafka-acls
docker compose logs mirror-maker
```

---

## 4. Проверить топики в primary Kafka

```bash
docker compose exec kafka1 kafka-topics.sh \
  --bootstrap-server kafka1:29092,kafka2:29092,kafka3:29092 \
  --command-config /opt/bitnami/kafka/config/certs/admin-ssl.properties \
  --list
```

Ожидаемые топики:

```text
shop.products.raw
shop.products.allowed
shop.products.rejected
shop.products.dlq
client.search.requests
client.recommendation.requests
analytics.recommendations
forbidden.products.commands
forbidden.products.state
```

---

## 5. Проверить топики в secondary Kafka

```bash
docker compose exec kafka2-1 kafka-topics.sh \
  --bootstrap-server kafka2-1:29092,kafka2-2:29092,kafka2-3:29092 \
  --command-config /opt/bitnami/kafka/config/certs/admin-ssl.properties \
  --list
```

Ожидаемо: те же базовые топики, плюс служебные топики MirrorMaker после запуска, например:

```text
heartbeats
checkpoints.internal
mm2-offset-syncs...
```

Названия служебных топиков могут отличаться.

---

## 6. Проверить репликацию и `min.insync.replicas`

Для primary:

```bash
docker compose exec kafka1 kafka-topics.sh \
  --bootstrap-server kafka1:29092,kafka2:29092,kafka3:29092 \
  --command-config /opt/bitnami/kafka/config/certs/admin-ssl.properties \
  --describe \
  --topic shop.products.allowed
```

Проверить:

```text
ReplicationFactor: 3
min.insync.replicas=2
```

Повторить для secondary:

```bash
docker compose exec kafka2-1 kafka-topics.sh \
  --bootstrap-server kafka2-1:29092,kafka2-2:29092,kafka2-3:29092 \
  --command-config /opt/bitnami/kafka/config/certs/admin-ssl.properties \
  --describe \
  --topic shop.products.allowed
```

---

## 7. Запустить сервисы приложения

В одном терминале:

```bash
docker compose --profile app up -d --build product-filter postgres-sink
```

Проверить:

```bash
docker compose ps product-filter postgres-sink
```

Смотреть логи:

```bash
docker compose logs -f product-filter
```

Во втором терминале:

```bash
docker compose logs -f postgres-sink
```

---

## 8. Добавить запрещённый товар

Через Compose job:

```bash
docker compose --profile jobs run --rm forbidden-cli
```

Или вручную с хоста:

```bash
go run ./services/forbidden-cli add \
  --brokers localhost:9092,localhost:9094,localhost:9096 \
  --tls \
  --tls-ca-cert ./configs/kafka/certs/ca.crt \
  --tls-client-cert ./configs/kafka/certs/admin.crt \
  --tls-client-key ./configs/kafka/certs/admin.key \
  --tls-server-name localhost \
  --product-id forbidden-001 \
  --reason "Seed product used to verify filtering"
```

Проверить список запрещённых товаров:

```bash
go run ./services/forbidden-cli list \
  --brokers localhost:9092,localhost:9094,localhost:9096 \
  --tls \
  --tls-ca-cert ./configs/kafka/certs/ca.crt \
  --tls-client-cert ./configs/kafka/certs/admin.crt \
  --tls-client-key ./configs/kafka/certs/admin.key \
  --tls-server-name localhost
```

Ожидаемо:

```text
Active forbidden products:
- forbidden-001 | active=true | reason=Seed product used to verify filtering
```

---

## 9. Отправить товары через SHOP API

Через Compose job:

```bash
docker compose --profile jobs run --rm shop-api
```

Или вручную с хоста:

```bash
go run ./services/shop-api \
  --brokers localhost:9092,localhost:9094,localhost:9096 \
  --tls \
  --tls-ca-cert ./configs/kafka/certs/ca.crt \
  --tls-client-cert ./configs/kafka/certs/shop-api.crt \
  --tls-client-key ./configs/kafka/certs/shop-api.key \
  --tls-server-name localhost \
  --file ./data/products.json
```

Ожидаемо в выводе:

```text
sent product watch-001 to shop.products.raw
sent product phone-001 to shop.products.raw
sent product forbidden-001 to shop.products.raw
```

---

## 10. Проверить работу фильтрации

В логах `product-filter` должно быть примерно:

```text
allowed product watch-001
allowed product phone-001
rejected forbidden product forbidden-001
```

Команда:

```bash
docker compose logs product-filter
```

Проверить allowed topic:

```bash
docker compose exec kafka1 kafka-console-consumer.sh \
  --bootstrap-server kafka1:29092,kafka2:29092,kafka3:29092 \
  --consumer.config /opt/bitnami/kafka/config/certs/admin-ssl.properties \
  --topic shop.products.allowed \
  --from-beginning \
  --timeout-ms 5000
```

Ожидаемо: есть `watch-001` и `phone-001`.

Проверить rejected topic:

```bash
docker compose exec kafka1 kafka-console-consumer.sh \
  --bootstrap-server kafka1:29092,kafka2:29092,kafka3:29092 \
  --consumer.config /opt/bitnami/kafka/config/certs/admin-ssl.properties \
  --topic shop.products.rejected \
  --from-beginning \
  --timeout-ms 5000
```

Ожидаемо: есть `forbidden-001`.

---

## 11. Проверить запись в PostgreSQL

Подключиться к PostgreSQL:

```bash
docker compose exec postgres psql -U marketplace -d marketplace
```

Выполнить:

```sql
SELECT product_id, name, category
FROM products
ORDER BY product_id;
```

Ожидаемо:

```text
phone-001
watch-001
```

Не должно быть:

```text
forbidden-001
```

Проверить отдельно:

```sql
SELECT COUNT(*)
FROM products
WHERE product_id = 'forbidden-001';
```

Ожидаемо:

```text
0
```

Выйти:

```sql
\q
```

---

## 12. Проверить CLIENT API search

Через Compose job:

```bash
docker compose --profile jobs run --rm client-api
```

Или вручную:

```bash
go run ./services/client-api search \
  --brokers localhost:9092,localhost:9094,localhost:9096 \
  --tls \
  --tls-ca-cert ./configs/kafka/certs/ca.crt \
  --tls-client-cert ./configs/kafka/certs/client-api.crt \
  --tls-client-key ./configs/kafka/certs/client-api.key \
  --tls-server-name localhost \
  --user-id user_001 \
  --query "умные часы"
```

Ожидаемо:

```text
Search results:
- watch-001 | Умные часы XYZ | Электроника | 4999.99 RUB
```

Также search-запрос должен попасть в Kafka topic:

```bash
docker compose exec kafka1 kafka-console-consumer.sh \
  --bootstrap-server kafka1:29092,kafka2:29092,kafka3:29092 \
  --consumer.config /opt/bitnami/kafka/config/certs/admin-ssl.properties \
  --topic client.search.requests \
  --from-beginning \
  --timeout-ms 5000
```

---

## 13. Проверить MirrorMaker 2

После отправки товаров и search-запроса подождать 10–20 секунд.

Проверить, что данные появились во втором Kafka-кластере.

Allowed products во secondary:

```bash
docker compose exec kafka2-1 kafka-console-consumer.sh \
  --bootstrap-server kafka2-1:29092,kafka2-2:29092,kafka2-3:29092 \
  --consumer.config /opt/bitnami/kafka/config/certs/admin-ssl.properties \
  --topic shop.products.allowed \
  --from-beginning \
  --timeout-ms 5000
```

Ожидаемо: `watch-001`, `phone-001`.

Search requests во secondary:

```bash
docker compose exec kafka2-1 kafka-console-consumer.sh \
  --bootstrap-server kafka2-1:29092,kafka2-2:29092,kafka2-3:29092 \
  --consumer.config /opt/bitnami/kafka/config/certs/admin-ssl.properties \
  --topic client.search.requests \
  --from-beginning \
  --timeout-ms 5000
```

Ожидаемо: событие поиска пользователя `user_001`.

---

## 14. Проверить ACL негативным тестом

Для полноценного негативного теста нужно создать отдельный client properties для `client-api.keystore.jks`, например:

```properties
security.protocol=SSL
ssl.truststore.location=/opt/bitnami/kafka/config/certs/kafka.truststore.jks
ssl.truststore.password=changeit
ssl.keystore.location=/opt/bitnami/kafka/config/certs/client-api.keystore.jks
ssl.keystore.password=changeit
ssl.key.password=changeit
```

Сохранить как:

```text
configs/kafka/certs/client-api-ssl.properties
```

Затем выполнить:

```bash
docker compose exec kafka1 kafka-console-producer.sh \
  --bootstrap-server kafka1:29092,kafka2:29092,kafka3:29092 \
  --producer.config /opt/bitnami/kafka/config/certs/client-api-ssl.properties \
  --topic shop.products.raw
```

Ожидаемо: ошибка авторизации, так как `client-api` не имеет `Write` на `shop.products.raw`.

---

## 15. Проверить отказоустойчивость Kafka

Остановить один брокер primary-кластера:

```bash
docker compose stop kafka3
```

Проверить описание топика:

```bash
docker compose exec kafka1 kafka-topics.sh \
  --bootstrap-server kafka1:29092,kafka2:29092 \
  --command-config /opt/bitnami/kafka/config/certs/admin-ssl.properties \
  --describe \
  --topic shop.products.allowed
```

Ожидаемо:

- кластер продолжает отвечать;
- часть partition может показать ISR из 2 реплик;
- запись должна продолжать работать, так как `min.insync.replicas=2`.

Вернуть брокер:

```bash
docker compose start kafka3
```

---

## 16. Быстрая проверка кода без Docker runtime

```bash
go test ./...
docker compose config
docker compose --profile app --profile jobs config
```

Ожидаемо: команды завершаются без ошибок.

---

## 17. Очистка окружения

```bash
docker compose down -v --remove-orphans
```

Или:

```bash
./scripts/reset.sh
```

---

## Что считается успешной проверкой

Минимально успешный результат:

1. Оба Kafka-кластера запущены.
2. Топики созданы с `ReplicationFactor: 3`.
3. TLS-команды работают через `admin-ssl.properties`.
4. ACL init завершается успешно.
5. `shop-api` отправляет товары.
6. `product-filter` пропускает разрешённые товары и отклоняет `forbidden-001`.
7. PostgreSQL содержит только разрешённые товары.
8. `client-api search` возвращает товар.
9. MirrorMaker переносит `shop.products.allowed` и `client.search.requests` во второй кластер.

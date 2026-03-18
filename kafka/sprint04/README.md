# Debezium CDC: PostgreSQL → Kafka + Monitoring (Prometheus/Grafana)

Решение поднимает полный стенд CDC (Change Data Capture):

- PostgreSQL как источник данных,
- Kafka как транспорт событий,
- Kafka Connect + Debezium Postgres Connector как CDC-движок,
- Prometheus + Grafana для мониторинга Kafka Connect.

Основной файл запуска: [`docker-compose.yaml`](docker-compose.yaml).

## Состав компонентов

### 1) PostgreSQL

- Сервис: `postgres`
- Конфиг логической репликации: [`postgres/custom-config.conf`](postgres/custom-config.conf)
- Инициализация БД (таблицы + тестовые данные): [`postgres/init.sql`](postgres/init.sql)

В БД `customers` создаются таблицы:

- `public.users`
- `public.orders`

### 2) Apache Kafka

- Сервисы: `kafka-0`, `kafka-1`, `kafka-2`
- ZooKeeper: `zookeeper`

Kafka хранит CDC-события из PostgreSQL в топиках Debezium.

### 3) Kafka Connect + Debezium

- Сервис: `kafka-connect`
- Docker-образ: [`kafka-connect/Dockerfile`](kafka-connect/Dockerfile)
- Конфиг коннектора: [`kafka-connect/connectors/postgres-source.json`](kafka-connect/connectors/postgres-source.json)
- Debezium плагин подключается из каталога [`confluent-hub-components/debezium-connector-postgres/`](confluent-hub-components/debezium-connector-postgres/)

Коннектор настроен отслеживать **только** таблицы:

- `public.users`
- `public.orders`

через параметр `table.include.list`.

### 4) Monitoring

- Prometheus: конфиг [`prometheus/prometheus.yml`](prometheus/prometheus.yml)
- Grafana:
  - Dockerfile: [`grafana/Dockerfile`](grafana/Dockerfile)
  - datasource provisioning: [`grafana/provisioning/datasources/all.yml`](grafana/provisioning/datasources/all.yml)
  - dashboards provisioning: [`grafana/provisioning/dashboards/all.yml`](grafana/provisioning/dashboards/all.yml)
  - dashboard: [`grafana/dashboards/connect.json`](grafana/dashboards/connect.json)

Prometheus снимает метрики Kafka Connect с endpoint `kafka-connect:9876` (JMX exporter).

### 5) Чтение событий из Kafka (код)

- Go consumer: [`consumer/main.go`](consumer/main.go)
- Go module: [`consumer/go.mod`](consumer/go.mod)

Программа читает события из:

- `postgres-cdc.public.users`
- `postgres-cdc.public.orders`

и выводит JSON в терминал.

## Как запустить

Из корня проекта (где находится [`docker-compose.yaml`](docker-compose.yaml)):

```bash
docker compose up -d --build
```

Проверить состояние сервисов:

```bash
docker compose ps
```

## Пошаговая проверка работоспособности

### Шаг 1. Убедиться, что PostgreSQL поднят и таблицы созданы

`init`-скрипт [`postgres/init.sql`](postgres/init.sql) выполняется автоматически при первом старте volume.

Проверка таблиц:

```bash
docker compose exec postgres psql -U postgres-user -d customers -c "\dt"
```

### Шаг 2. Зарегистрировать Debezium Connector

Создать коннектор из файла [`kafka-connect/connectors/postgres-source.json`](kafka-connect/connectors/postgres-source.json):

```powershell
$body = Get-Content -Raw -Path "kafka-connect/connectors/postgres-source.json"
Invoke-RestMethod -Method Post -Uri "http://127.0.0.1:8083/connectors" -ContentType "application/json" -Body $body
```

Альтернатива через настоящий curl в Windows:

```powershell
curl.exe -X POST "http://127.0.0.1:8083/connectors" -H "Content-Type: application/json" --data-binary "@kafka-connect/connectors/postgres-source.json"
```

### Шаг 3. Проверить статус коннектора

```powershell
Invoke-RestMethod -Method Get -Uri "http://127.0.0.1:8083/connectors/postgres-source/status"
```

Ожидаемо:

- `connector.state = RUNNING`
- `tasks[0].state = RUNNING`

### Шаг 4. Проверить наличие CDC-топиков

```bash
docker compose exec kafka-0 kafka-topics.sh --bootstrap-server kafka-0:9092 --list
```
или посмотреть в [Kafka UI](http://localhost:8080/ui/clusters/zookeeper-cluster/all-topics?perPage=25&hideInternal=true)

Должны появиться:

- `postgres-cdc.public.users`
- `postgres-cdc.public.orders`

### Шаг 5. Прочитать события кодом на Go

Скачать зависимости и собрать:

```powershell
Set-Location consumer; go mod tidy; go build -o read-cdc.exe .
```

Запустить consumer:

```powershell
Set-Location consumer; .\read-cdc.exe
```

В терминале будут выводиться CDC-события (snapshot + новые изменения).

### Шаг 6. Сгенерировать дополнительные изменения в PostgreSQL

```bash
docker compose exec postgres psql -U postgres-user -d customers -c "INSERT INTO users (name, email) VALUES ('Test User', 'test@example.com');"
docker compose exec postgres psql -U postgres-user -d customers -c "INSERT INTO orders (user_id, product_name, quantity) VALUES (1, 'Product X', 10);"
```

События по этим изменениям появятся в запущенном [`consumer/main.go`](consumer/main.go).

## Тестовые данные

В [`postgres/init.sql`](postgres/init.sql) уже добавлены записи:

- Пользователи: John Doe, Jane Smith, Alice Johnson, Bob Brown
- Заказы: Product A..E

## Мониторинг: Prometheus и Grafana

### Prometheus

- UI: <http://127.0.0.1:9090>
- Job для Kafka Connect описан в [`prometheus/prometheus.yml`](prometheus/prometheus.yml)

### Grafana

- UI: <http://127.0.0.1:3000>
- Dashboard `Kafka Connect Overview-0` загружается из [`grafana/dashboards/connect.json`](grafana/dashboards/connect.json)

На дашборде есть графики для:

- состояния коннектора и задач (`running/failed/paused`),
- скорости чтения/записи записей (`poll/write rate`),
- метрик производительности передачи данных.

## Полезные endpoint'ы

- Kafka UI: <http://127.0.0.1:8080>
- Schema Registry: <http://127.0.0.1:8081>
- Kafka Connect REST: <http://127.0.0.1:8083>
- Prometheus: <http://127.0.0.1:9090>
- Grafana: <http://127.0.0.1:3000>
- PostgreSQL: `127.0.0.1:5432`

## Остановка

Остановить окружение:

```bash
docker compose down
```

Полный сброс (включая volume):

```bash
docker compose down -v
```

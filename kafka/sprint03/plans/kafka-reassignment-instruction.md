# Kafka: Partition Reassignment и диагностика сбоя брокера

## 1) Предусловия

- Docker Desktop запущен.
- В каталоге проекта есть [`docker-compose.yml`](docker-compose.yml).
- Кластер поднят:

### CMD
```bat
docker compose up -d
docker compose ps
```

### PowerShell (альтернатива)
```powershell
docker compose up -d
docker compose ps
```

Ожидается, что контейнеры `kafka-1`, `kafka-2`, `kafka-3`, `zookeeper` в состоянии `Up`.

---

## 2) Что подготовлено

### Основные `.cmd` скрипты

- [`scripts/01-create-topic.cmd`](scripts/01-create-topic.cmd)
- [`scripts/02-describe-topic-before.cmd`](scripts/02-describe-topic-before.cmd)
- [`scripts/03-generate-reassignment-json.cmd`](scripts/03-generate-reassignment-json.cmd)
- [`scripts/04-execute-reassignment.cmd`](scripts/04-execute-reassignment.cmd)
- [`scripts/05-verify-reassignment.cmd`](scripts/05-verify-reassignment.cmd)
- [`scripts/06-describe-topic-after.cmd`](scripts/06-describe-topic-after.cmd)
- [`scripts/07-stop-broker-kafka-1.cmd`](scripts/07-stop-broker-kafka-1.cmd)
- [`scripts/08-check-topics-during-failure.cmd`](scripts/08-check-topics-during-failure.cmd)
- [`scripts/09-start-broker-kafka-1.cmd`](scripts/09-start-broker-kafka-1.cmd)
- [`scripts/10-check-isr-recovery.cmd`](scripts/10-check-isr-recovery.cmd)
- [`scripts/11-run-all.cmd`](scripts/11-run-all.cmd)

### Файлы данных

- [`assets/topics-to-move.json`](assets/topics-to-move.json)
- `assets/reassignment.json` (генерируется скриптом [`scripts/03-generate-reassignment-json.cmd`](scripts/03-generate-reassignment-json.cmd))

---

## 3) Выполнение задания (по пунктам)

Открой терминал в корне проекта и запускай команды по порядку.

### Шаг 1. Создать топик `balanced_topic` (8 партиций, RF=3)

```bat
scripts\01-create-topic.cmd
```

PowerShell:
```powershell
.\scripts\01-create-topic.cmd
```

### Шаг 2. Посмотреть текущее распределение партиций

```bat
scripts\02-describe-topic-before.cmd
```

PowerShell:
```powershell
.\scripts\02-describe-topic-before.cmd
```

### Шаг 3. Сгенерировать `reassignment.json`

```bat
scripts\03-generate-reassignment-json.cmd
```

PowerShell:
```powershell
.\scripts\03-generate-reassignment-json.cmd
```

Результат: появляется файл `assets/reassignment.json`.

### Шаг 4. Выполнить перераспределение партиций

```bat
scripts\04-execute-reassignment.cmd
```

PowerShell:
```powershell
.\scripts\04-execute-reassignment.cmd
```

### Шаг 5. Проверить статус перераспределения

```bat
scripts\05-verify-reassignment.cmd
```

PowerShell:
```powershell
.\scripts\05-verify-reassignment.cmd
```

### Шаг 6. Убедиться, что конфигурация изменилась

```bat
scripts\06-describe-topic-after.cmd
```

PowerShell:
```powershell
.\scripts\06-describe-topic-after.cmd
```

Сравни вывод шага 2 и шага 6: должны измениться назначения лидеров/реплик (в зависимости от сгенерированного плана).

---

## 4) Моделирование сбоя брокера

### Шаг 7.1. Остановить `kafka-1`

```bat
scripts\07-stop-broker-kafka-1.cmd
```

PowerShell:
```powershell
.\scripts\07-stop-broker-kafka-1.cmd
```

### Шаг 7.2. Проверить состояние топиков после сбоя

```bat
scripts\08-check-topics-during-failure.cmd
```

PowerShell:
```powershell
.\scripts\08-check-topics-during-failure.cmd
```

Ожидаемо увидеть уменьшенный ISR и/или under-replicated partitions.

### Шаг 7.3. Запустить `kafka-1` обратно

```bat
scripts\09-start-broker-kafka-1.cmd
```

PowerShell:
```powershell
.\scripts\09-start-broker-kafka-1.cmd
```

### Шаг 7.4. Проверить восстановление синхронизации реплик

```bat
scripts\10-check-isr-recovery.cmd
```

PowerShell:
```powershell
.\scripts\10-check-isr-recovery.cmd
```

Если ISR еще не полный — подожди 15–60 секунд и повтори шаг 7.4.

---

## 5) Быстрый прогон основной части (без сбоя)

```bat
scripts\11-run-all.cmd
```

PowerShell:
```powershell
.\scripts\11-run-all.cmd
```

Этот скрипт выполняет шаги 1–6 подряд.

---

## 6) Полезные ручные команды (опционально)

### Проверить состояние контейнеров

CMD/PowerShell:
```text
docker compose ps
```

### Посмотреть `balanced_topic` напрямую

CMD:
```bat
docker compose exec -T kafka-1 kafka-topics --bootstrap-server kafka-1:29092 --describe --topic balanced_topic
```

PowerShell:
```powershell
docker compose exec -T kafka-1 kafka-topics --bootstrap-server kafka-1:29092 --describe --topic balanced_topic
```


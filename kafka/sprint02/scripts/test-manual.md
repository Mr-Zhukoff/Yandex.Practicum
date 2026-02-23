# Руководство по ручному тестированию

## Подготовка

Убедитесь, что запущены:
1. Kafka инфраструктура: `docker-compose up -d`
2. Процессор: `go run cmd/processor/main.go`
3. Консьюмер (опционально): `go run cmd/consumer/main.go`

## Тестовые команды

### 1. Добавление запрещённых слов

```bash
docker exec -it kafka kafka-console-producer --topic censor-actions --bootstrap-server localhost:9093 --property "parse.key=true" --property "key.separator=:"
```

Введите:
```
global:{"word":"badword","action":"add","timestamp":1708368000}
global:{"word":"spam","action":"add","timestamp":1708368001}
global:{"word":"offensive","action":"add","timestamp":1708368002}
```

### 2. Отправка обычных сообщений

```bash
docker exec -it kafka kafka-console-producer --topic messages --bootstrap-server localhost:9093 --property "parse.key=true" --property "key.separator=:"
```

Введите:
```
alice:{"id":"msg1","from":"alice","to":"bob","content":"Hello Bob, how are you?","timestamp":1708368100}
bob:{"id":"msg2","from":"bob","to":"alice","content":"Hi Alice, I'm doing great!","timestamp":1708368101}
```

### 3. Отправка сообщений с цензурой

```bash
docker exec -it kafka kafka-console-producer --topic messages --bootstrap-server localhost:9093 --property "parse.key=true" --property "key.separator=:"
```

Введите:
```
charlie:{"id":"msg3","from":"charlie","to":"bob","content":"This is a badword message!","timestamp":1708368200}
alice:{"id":"msg4","from":"alice","to":"bob","content":"No spam here, just testing","timestamp":1708368201}
```

### 4. Блокировка пользователей

```bash
docker exec -it kafka kafka-console-producer --topic blocked_users --bootstrap-server localhost:9093 --property "parse.key=true" --property "key.separator=:"
```

Введите:
```
bob:{"blocker_id":"bob","blocked_id":"charlie","action":"block","timestamp":1708368300}
alice:{"blocker_id":"alice","blocked_id":"dave","action":"block","timestamp":1708368301}
```

### 5. Тест заблокированных сообщений

```bash
docker exec -it kafka kafka-console-producer --topic messages --bootstrap-server localhost:9093 --property "parse.key=true" --property "key.separator=:"
```

Введите:
```
charlie:{"id":"msg5","from":"charlie","to":"bob","content":"Can you see this?","timestamp":1708368400}
alice:{"id":"msg6","from":"alice","to":"bob","content":"Hello from Alice","timestamp":1708368401}
```

Результат: msg5 не пройдёт (заблокирован), msg6 пройдёт

### 6. Разблокировка

```bash
docker exec -it kafka kafka-console-producer --topic blocked_users --bootstrap-server localhost:9093 --property "parse.key=true" --property "key.separator=:"
```

Введите:
```
bob:{"blocker_id":"bob","blocked_id":"charlie","action":"unblock","timestamp":1708368500}
```

### 7. Просмотр результатов

```bash
docker exec -it kafka kafka-console-consumer --topic filtered_messages --bootstrap-server localhost:9093 --from-beginning
```

## Проверка топиков

Список всех топиков:
```bash
docker exec -it kafka kafka-topics --list --bootstrap-server localhost:9093
```

Информация о топике:
```bash
docker exec -it kafka kafka-topics --describe --topic messages --bootstrap-server localhost:9093
```

Чтение из топика:
```bash
docker exec -it kafka kafka-console-consumer --topic messages --bootstrap-server localhost:9093 --from-beginning --max-messages 10
```

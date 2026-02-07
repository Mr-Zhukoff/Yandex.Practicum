# Kafka sprint 1
Запускаем Docker Compose, чтобы создать и запустить контейнеры:

`docker-compose up -d` 

Проверяем успешность запуска. Убедитесь, что контейнеры работают:

`docker ps` 

Вы должны увидеть пять работающих контейнеров:
 - 1 для Zookeeper 
 - 3 для Kafka
 - 1 для Kafka-UI

Для создания топика топик с 3 партициями и 2 репликами выполните следующую команду

```
docker exec sprint01-kafka-1-1 kafka-topics --create \
  --topic my-topic \
  --partitions 3 \
  --replication-factor 2 \
  --bootstrap-server kafka-1:29092
```

Проверить можно командой describe

`docker exec sprint01-kafka-1-1 kafka-topics --describe --topic my-topic --bootstrap-server kafka-1:29092
`


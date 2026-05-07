# # Задание 1. Развёртывание и настройка Kafka-кластера

Документ описывает базовый подход для Linux-серверов, когда к узлам есть только SSH-доступ и нет прямого доступа к UI.

## 1. Что развернуто

- Kafka-кластер из 3 брокеров:
  - [`kafka-rc1a`](rc1a-gi7jh7m9q3te5h7o.mdb.yandexcloud.net)
  - [`kafka-rc1b`](rc1b-od1poa2euekqn2d1.mdb.yandexcloud.net)
  - [`kafka-rc1d`](rc1d-ktba71vfmb0m0sp8.mdb.yandexcloud.net)
  - [`ZooKeeper-rc1a`](rc1a-e5ebqvb1cbeje5mo.mdb.yandexcloud.net)
  - [`ZooKeeper-rc1b`](rc1b-1ja1581hvi2ro8ui.mdb.yandexcloud.net)
  - [`ZooKeeper-rc1d`](rc1d-1ocdvbgjferptg7p.mdb.yandexcloud.net)
- [`Schema Registry`](docker-compose.yml)
- [`Kafka UI`](https://ui-c9qe85o8a5tfbat5th20.kafka.yandexcloud.net/)

## 2. Рекомендуемые ресурсы (production-like)

Для каждого Kafka-брокера:

- CPU: 4 vCPU (минимум 2)
- RAM: 64 GB 
- Disk: SSD/NVMe, от 100 GB, отдельный диск/том под логи Kafka
- Файловая система: XFS/ext4, `noatime`

ZooKeeper столько же нод сколько у кластера
- CPU: 2-4 vCPU 
- RAM: 4 GB 
- Disk: SSD/NVMe, от 512 GB

Schema Registry 
- CPU: 2-4 vCPU 
- RAM: 2 GB 
- Disk: SSD/NVMe, 128 GB

## 3. Создание пользователя через Yandex Cloud (CLI)

```bash
yc managed-kafka user create kafka_admin --cluster-name kafka502 --password P@ssw0rd --permission topic=*,role=admin,allow_host=rc1a-gi7jh7m9q3te5h7o.mdb.yandexcloud.net,allow_host=rc1b-od1poa2euekqn2d1.mdb.yandexcloud.net,allow_host=rc1d-ktba71vfmb0m0sp8.mdb.yandexcloud.net
```

## 4. Настройка топиков через Yandex Cloud (CLI)

```bash
yc managed-kafka topic create user-events --cluster-name kafka502 --partitions 3 --replication-factor 3 --cleanup-policy compact_and_delete --delete-retention-ms 86400000 --segment-bytes 10485760
```

## 5. Настройка Schema Registry

Включен Managed Schema Registry

```bash
curl -s -X POST -H "Content-Type: application/vnd.schemaregistry.v1+json" --data-binary @payload.json http://rc1a-gi7jh7m9q3te5h7o.mdb.yandexcloud.net:8081/subjects/user-events-value/versions
```

## 6. Запуск приложений

Запуск продюсера

```bash
 $env:KAFKA_BROKERS='rc1a-gi7jh7m9q3te5h7o.mdb.yandexcloud.net:9091,rc1b-od1poa2euekqn2d1.mdb.yandexcloud.net:9091,rc1d-ktba71vfmb0m0sp8.mdb.yandexcloud.net:9091'; $env:KAFKA_USERNAME='producer'; $env:KAFKA_PASSWORD='P@ssw0rd'; $env:KAFKA_SASL_MECHANISM='SCRAM-SHA-512'; $env:KAFKA_CA_CERT_PATH='C:\Users\Zhuko\.kafka\YandexInternalRootCA.crt'; $env:KAFKA_TOPIC='user-events'; go run ./cmd/producer/main.go
```

Завпуск консьюмера

```bash
 $env:KAFKA_BROKERS='rc1a-gi7jh7m9q3te5h7o.mdb.yandexcloud.net:9091,rc1b-od1poa2euekqn2d1.mdb.yandexcloud.net:9091,rc1d-ktba71vfmb0m0sp8.mdb.yandexcloud.net:9091'; $env:KAFKA_USERNAME='consumer'; $env:KAFKA_PASSWORD='P@ssw0rd'; $env:KAFKA_SASL_MECHANISM='SCRAM-SHA-512'; $env:KAFKA_CA_CERT_PATH='C:\Users\Zhuko\.kafka\YandexInternalRootCA.crt'; $env:KAFKA_TOPIC='user-events'; go run ./cmd/consumer/main.go
```

## 7 Логи

Скриншоты в папке screenshots

Создание пользователя
```
PS C:\Users\Zhuko> yc managed-kafka user create kafka_admin --cluster-name kafka502 --password P@ssw0rd --permission topic=*,role=admin,allow_host=rc1a-gi7jh7m9q3te5h7o.mdb.yandexcloud.net,allow_host=rc1b-od1poa2euekqn2d1.mdb.yandexcloud.net,allow_host=rc1d-ktba71vfmb0m0sp8.mdb.yandexcloud.net
WARNING: Cannot connect to YC tool initialization service. Network connectivity to the service is required for cli version control. In case you are using yc in an isolated environment, you may turn off this warning by setting env YC_CLI_INITIALIZATION_SILENCE=true

done (42s)
name: kafka_admin
cluster_id: c9qe85o8a5tfbat5th20
permissions:
  - topic_name: '*'
    role: ACCESS_ROLE_ADMIN
    allow_hosts:
      - rc1a-gi7jh7m9q3te5h7o.mdb.yandexcloud.net
      - rc1b-od1poa2euekqn2d1.mdb.yandexcloud.net
      - rc1d-ktba71vfmb0m0sp8.mdb.yandexcloud.net
```
Создание топика
```
PS C:\Users\Zhuko> yc managed-kafka topic create user-events --cluster-name kafka502 --partitions 3 --replication-factor 3 --cleanup-policy compact_and_delete --delete-retention-ms 86400000 --segment-bytes 10485760
WARNING: Cannot connect to YC tool initialization service. Network connectivity to the service is required for cli version control. In case you are using yc in an isolated environment, you may turn off this warning by setting env YC_CLI_INITIALIZATION_SILENCE=true

done (39s)
name: user-events
cluster_id: c9qe85o8a5tfbat5th20
partitions: "3"
replication_factor: "3"
topic_config_3:
  cleanup_policy: CLEANUP_POLICY_COMPACT_AND_DELETE
  delete_retention_ms: "86400000"
  segment_bytes: "10485760"
```
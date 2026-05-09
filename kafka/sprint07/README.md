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

## 7. Часть 2 Интеграция с NiFi

Интеграция настроена так:
NiFi читает файлы *.csv из папки nifi_data и отправляет их в топик users в yandex kafka в облаке, затем консьюмер на Go читает их из топика users.

Что было сделано:
- Консьюмер обновлён для чтения из двух топиков: `user-events` и `users` (параллельное чтение).
- Добавлена схема для топика `users`: [`schemas/register-users-value.json`](schemas/register-users-value.json).
- Для `PublishKafkaRecord_2_0` в NiFi зафиксирован важный нюанс: топик `users` с политикой `compact/compact_and_delete` требует обязательный key у сообщений (например, поле `id`).
- Для подключения к Managed Kafka в NiFi используется `SASL_SSL + SCRAM-SHA-512` и truststore с CA-сертификатом.



## 8 Логи

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

Логи NiFi

```
	request.timeout.ms = 30000
	retries = 0
	retry.backoff.ms = 100
	sasl.client.callback.handler.class = null
	sasl.jaas.config = [hidden]
	sasl.kerberos.kinit.cmd = /usr/bin/kinit
	sasl.kerberos.min.time.before.relogin = 60000
	sasl.kerberos.service.name = null
	sasl.kerberos.ticket.renew.jitter = 0.05
	sasl.kerberos.ticket.renew.window.factor = 0.8
	sasl.login.callback.handler.class = null
	sasl.login.class = null
	sasl.login.refresh.buffer.seconds = 300
	sasl.login.refresh.min.period.seconds = 60
	sasl.login.refresh.window.factor = 0.8
	sasl.login.refresh.window.jitter = 0.05
	sasl.mechanism = SCRAM-SHA-512
	security.protocol = SASL_SSL
	send.buffer.bytes = 131072
	ssl.cipher.suites = null
	ssl.enabled.protocols = [TLSv1.2, TLSv1.1, TLSv1]
	ssl.endpoint.identification.algorithm = https
	ssl.key.password = null
	ssl.keymanager.algorithm = SunX509
	ssl.keystore.location = null
	ssl.keystore.password = null
	ssl.keystore.type = JKS
	ssl.protocol = TLS
	ssl.provider = null
	ssl.secure.random.implementation = null
	ssl.trustmanager.algorithm = PKIX
	ssl.truststore.location = /opt/certs/yandex-truststore.p12
	ssl.truststore.password = [hidden]
	ssl.truststore.type = PKCS12
	transaction.timeout.ms = 60000
	transactional.id = null
	value.serializer = class org.apache.kafka.common.serialization.ByteArraySerializer

2026-05-09 20:24:11,510 INFO [Timer-Driven Process Thread-9] o.a.k.c.s.authenticator.AbstractLogin Successfully logged in.

2026-05-09 20:24:11,515 INFO [Timer-Driven Process Thread-9] o.a.kafka.common.utils.AppInfoParser Kafka version : 2.0.0

2026-05-09 20:24:11,515 INFO [Timer-Driven Process Thread-9] o.a.kafka.common.utils.AppInfoParser Kafka commitId : 3402a8361b734732

2026-05-09 20:24:11,755 INFO [kafka-producer-network-thread | producer-6] org.apache.kafka.clients.Metadata Cluster ID: uBMWR9nFSweLdCLF1hxDHw

2026-05-09 20:24:15,394 INFO [pool-7-thread-1] o.a.n.c.r.WriteAheadFlowFileRepository Initiating checkpoint of FlowFile Repository

2026-05-09 20:24:15,396 INFO [pool-7-thread-1] o.a.n.wali.SequentialAccessWriteAheadLog Checkpointed Write-Ahead Log with 0 Records and 0 Swap Files in 2 milliseconds (Stop-the-world time = 0 milliseconds), max Transaction ID 11

2026-05-09 20:24:15,396 INFO [pool-7-thread-1] o.a.n.c.r.WriteAheadFlowFileRepository Successfully checkpointed FlowFile Repository with 0 records in 2 milliseconds

2026-05-09 20:24:33,570 INFO [Cleanup Archive for default] o.a.n.c.repository.FileSystemRepository Successfully deleted 0 files (0 bytes) from archive

2026-05-09 20:24:33,570 INFO [Cleanup Archive for default] o.a.n.c.repository.FileSystemRepository Archive cleanup completed for container default; will now allow writing to this container. Bytes used = 129.27 GB, bytes free = 877.59 GB, capacity = 1,006.85 GB

2026-05-09 20:24:33,620 INFO [Write-Ahead Local State Provider Maintenance] org.wali.MinimalLockingWriteAheadLog org.wali.MinimalLockingWriteAheadLog@7957eec3 checkpointed with 2 Records and 0 Swap Files in 2 milliseconds (Stop-the-world time = 0 milliseconds, Clear Edit Logs time = 0 millis), max Transaction ID 1
```

Лог консьюмера

```
PS S:\GitHub\Yandex.Practicum\kafka\sprint07\app> $env:KAFKA_BROKERS='rc1a-gi7jh7m9q3te5h7o.mdb.yandexcloud.net:9091,rc1b-od1poa2euekqn2d1.mdb.yandexcloud.net:9091,rc1d-ktba71vfmb0m0sp8.mdb.yandexcloud.net:9091'; $env:KAFKA_USERNAME='consumer'; $env:KAFKA_PASSWORD='P@ssw0rd'; $env:KAFKA_SASL_MECHANISM='SCRAM-SHA-512'; $env:KAFKA_CA_CERT_PATH='s:\GitHub\Yandex.Practicum\kafka\sprint07\.certs\YandexInternalRootCA.crt'; go run ./cmd/consumer/main.go
consumer[users]: received topic=users partition=0 offset=0 key=1 value={"id":1,"name":"Екатерина","age":21,"email":"kate@yandex.ru"}
consumer[users]: received topic=users partition=0 offset=1 key=5 value={"id":5,"name":"Илья","age":25,"email":"ilya@yandex.ru"}
consumer[users]: received topic=users partition=0 offset=2 key=1 value={"id":1,"name":"Екатерина","age":21,"email":"kate@yandex.ru"}
consumer[users]: received topic=users partition=0 offset=3 key=5 value={"id":5,"name":"Илья","age":25,"email":"ilya@yandex.ru"}
consumer[users]: received topic=users partition=1 offset=0 key=4 value={"id":4,"name":"Алексей","age":26,"email":"alex@yandex.ru"}
consumer[users]: received topic=users partition=1 offset=1 key=6 value={"id":6,"name":"Виктория","age":22,"email":"vika@yandex.ru"}
consumer[users]: received topic=users partition=1 offset=2 key=4 value={"id":4,"name":"Алексей","age":26,"email":"alex@yandex.ru"}
consumer[users]: received topic=users partition=1 offset=3 key=6 value={"id":6,"name":"Виктория","age":22,"email":"vika@yandex.ru"}
consumer[users]: received topic=users partition=2 offset=0 key=2 value={"id":2,"name":"Никита","age":26,"email":"nikita@yandex.ru"}
consumer[users]: received topic=users partition=2 offset=1 key=3 value={"id":3,"name":"Майя","age":21,"email":"maya@yandex.ru"}
consumer[users]: received topic=users partition=2 offset=2 key=2 value={"id":2,"name":"Никита","age":26,"email":"nikita@yandex.ru"}
consumer[users]: received topic=users partition=2 offset=3 key=3 value={"id":3,"name":"Майя","age":21,"email":"maya@yandex.ru"}
2026/05/09 23:26:36 consumer[user-events]: stop reading: fetching message: context deadline exceeded
2026/05/09 23:26:36 consumer[users]: stop reading: fetching message: context deadline exceeded
```
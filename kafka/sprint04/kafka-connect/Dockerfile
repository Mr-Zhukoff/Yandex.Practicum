FROM confluentinc/cp-kafka-connect:7.7.1
# Подключить Debezium PostgreSQL Source Connector из официального образа Debezium
COPY /kafka/connect/debezium-connector-postgres /usr/share/java/debezium-connector-postgres
# Скопировать JMX Exporter в докер образ
COPY jmx_prometheus_javaagent-0.15.0.jar /opt/
# Скопировать настройки JMX Exporter в докер образ
COPY kafka-connect.yml /opt/

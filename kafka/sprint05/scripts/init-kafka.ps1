$ErrorActionPreference = 'Stop'

$cmdBase = "docker compose exec -T kafka-1"

$adminCfgHostDir = Join-Path $PSScriptRoot '..\certs\clients'
$adminCfgHostPath = Join-Path $adminCfgHostDir 'admin-client.properties'
$adminCfgPath = '/etc/kafka/secrets/clients/admin-client.properties'

New-Item -ItemType Directory -Force -Path $adminCfgHostDir | Out-Null
@"
security.protocol=SSL
ssl.truststore.location=/etc/kafka/secrets/clients/admin.truststore.jks
ssl.truststore.password=changeit
ssl.keystore.location=/etc/kafka/secrets/clients/admin.keystore.jks
ssl.keystore.password=changeit
ssl.key.password=changeit
"@ | Set-Content -Path $adminCfgHostPath -Encoding ascii

Invoke-Expression "$cmdBase kafka-topics --bootstrap-server kafka-1:9092 --create --if-not-exists --topic topic-1 --partitions 3 --replication-factor 3 --command-config $adminCfgPath"
Invoke-Expression "$cmdBase kafka-topics --bootstrap-server kafka-1:9092 --create --if-not-exists --topic topic-2 --partitions 3 --replication-factor 3 --command-config $adminCfgPath"

Invoke-Expression "$cmdBase kafka-acls --bootstrap-server kafka-1:9092 --add --allow-principal 'User:CN=producer' --operation Write --topic topic-1 --command-config $adminCfgPath"
Invoke-Expression "$cmdBase kafka-acls --bootstrap-server kafka-1:9092 --add --allow-principal 'User:CN=producer' --operation Write --topic topic-2 --command-config $adminCfgPath"
Invoke-Expression "$cmdBase kafka-acls --bootstrap-server kafka-1:9092 --add --allow-principal 'User:CN=producer' --operation Describe --topic topic-1 --command-config $adminCfgPath"
Invoke-Expression "$cmdBase kafka-acls --bootstrap-server kafka-1:9092 --add --allow-principal 'User:CN=producer' --operation Describe --topic topic-2 --command-config $adminCfgPath"

Invoke-Expression "$cmdBase kafka-acls --bootstrap-server kafka-1:9092 --add --allow-principal 'User:CN=consumer' --operation Read --topic topic-1 --command-config $adminCfgPath"
Invoke-Expression "$cmdBase kafka-acls --bootstrap-server kafka-1:9092 --add --allow-principal 'User:CN=consumer' --operation Describe --topic topic-1 --command-config $adminCfgPath"
Invoke-Expression "$cmdBase kafka-acls --bootstrap-server kafka-1:9092 --add --allow-principal 'User:CN=consumer' --operation Read --group consumer-group-1 --command-config $adminCfgPath"

Write-Host 'Topics and ACLs are configured.'


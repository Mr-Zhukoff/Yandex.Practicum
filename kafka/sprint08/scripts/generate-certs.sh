#!/usr/bin/env bash
set -euo pipefail

CERT_DIR="${CERT_DIR:-./configs/kafka/certs}"
PASSWORD="${CERT_PASSWORD:-changeit}"
VALIDITY_DAYS="${VALIDITY_DAYS:-3650}"

mkdir -p "${CERT_DIR}"
rm -f "${CERT_DIR}"/*.jks "${CERT_DIR}"/*.crt "${CERT_DIR}"/*.csr "${CERT_DIR}"/*.srl

echo "Generating CA..."
openssl req -new -x509 \
  -keyout "${CERT_DIR}/ca.key" \
  -out "${CERT_DIR}/ca.crt" \
  -days "${VALIDITY_DAYS}" \
  -nodes \
  -subj "/CN=marketplace-kafka-ca"

create_keystore() {
  local name="$1"
  local cn="$2"
  local san="$3"

  echo "Generating keystore for ${name}..."
  keytool -genkeypair \
    -alias "${name}" \
    -keyalg RSA \
    -keysize 2048 \
    -validity "${VALIDITY_DAYS}" \
    -keystore "${CERT_DIR}/${name}.keystore.jks" \
    -storepass "${PASSWORD}" \
    -keypass "${PASSWORD}" \
    -dname "CN=${cn}" \
    -ext "SAN=${san}" \
    -storetype JKS \
    -noprompt

  keytool -certreq \
    -alias "${name}" \
    -file "${CERT_DIR}/${name}.csr" \
    -keystore "${CERT_DIR}/${name}.keystore.jks" \
    -storepass "${PASSWORD}" \
    -keypass "${PASSWORD}"

  openssl x509 -req \
    -CA "${CERT_DIR}/ca.crt" \
    -CAkey "${CERT_DIR}/ca.key" \
    -in "${CERT_DIR}/${name}.csr" \
    -out "${CERT_DIR}/${name}.crt" \
    -days "${VALIDITY_DAYS}" \
    -CAcreateserial \
    -extfile <(printf "subjectAltName=%s" "${san}")

  keytool -importcert \
    -alias CARoot \
    -file "${CERT_DIR}/ca.crt" \
    -keystore "${CERT_DIR}/${name}.keystore.jks" \
    -storepass "${PASSWORD}" \
    -noprompt

  keytool -importcert \
    -alias "${name}" \
    -file "${CERT_DIR}/${name}.crt" \
    -keystore "${CERT_DIR}/${name}.keystore.jks" \
    -storepass "${PASSWORD}" \
    -noprompt

  keytool -importkeystore \
    -srckeystore "${CERT_DIR}/${name}.keystore.jks" \
    -destkeystore "${CERT_DIR}/${name}.p12" \
    -srcstoretype JKS \
    -deststoretype PKCS12 \
    -srcstorepass "${PASSWORD}" \
    -deststorepass "${PASSWORD}" \
    -srcalias "${name}" \
    -destalias "${name}" \
    -srckeypass "${PASSWORD}" \
    -destkeypass "${PASSWORD}" \
    -noprompt

  openssl pkcs12 \
    -in "${CERT_DIR}/${name}.p12" \
    -nodes \
    -nocerts \
    -passin "pass:${PASSWORD}" \
    -out "${CERT_DIR}/${name}.key"
}

create_keystore "kafka1" "kafka1" "DNS:kafka1,DNS:localhost,IP:127.0.0.1"
create_keystore "kafka2" "kafka2" "DNS:kafka2,DNS:localhost,IP:127.0.0.1"
create_keystore "kafka3" "kafka3" "DNS:kafka3,DNS:localhost,IP:127.0.0.1"
create_keystore "kafka2-1" "kafka2-1" "DNS:kafka2-1,DNS:localhost,IP:127.0.0.1"
create_keystore "kafka2-2" "kafka2-2" "DNS:kafka2-2,DNS:localhost,IP:127.0.0.1"
create_keystore "kafka2-3" "kafka2-3" "DNS:kafka2-3,DNS:localhost,IP:127.0.0.1"
create_keystore "admin" "admin" "DNS:admin,DNS:localhost,IP:127.0.0.1"
create_keystore "shop-api" "shop-api" "DNS:shop-api,DNS:localhost,IP:127.0.0.1"
create_keystore "client-api" "client-api" "DNS:client-api,DNS:localhost,IP:127.0.0.1"
create_keystore "product-filter" "product-filter" "DNS:product-filter,DNS:localhost,IP:127.0.0.1"
create_keystore "postgres-sink" "postgres-sink" "DNS:postgres-sink,DNS:localhost,IP:127.0.0.1"
create_keystore "mirror-maker" "mirror-maker" "DNS:mirror-maker,DNS:localhost,IP:127.0.0.1"

echo "Generating shared truststore..."
keytool -importcert \
  -alias CARoot \
  -file "${CERT_DIR}/ca.crt" \
  -keystore "${CERT_DIR}/kafka.truststore.jks" \
  -storepass "${PASSWORD}" \
  -storetype JKS \
  -noprompt

cp "${CERT_DIR}/kafka.truststore.jks" "${CERT_DIR}/client.truststore.jks"

cat > "${CERT_DIR}/admin-ssl.properties" <<EOF
security.protocol=SSL
ssl.truststore.location=/opt/bitnami/kafka/config/certs/kafka.truststore.jks
ssl.truststore.password=${PASSWORD}
ssl.keystore.location=/opt/bitnami/kafka/config/certs/admin.keystore.jks
ssl.keystore.password=${PASSWORD}
ssl.key.password=${PASSWORD}
EOF

echo "Certificates generated in ${CERT_DIR}"

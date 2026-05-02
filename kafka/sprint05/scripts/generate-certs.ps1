$ErrorActionPreference = 'Stop'

$base = Join-Path $PSScriptRoot '..'
$certBase = Join-Path $base 'certs'
$brokersDir = Join-Path $certBase 'brokers'
$clientsDir = Join-Path $certBase 'clients'
$tmpDir = Join-Path $certBase 'tmp'

$storePass = 'changeit'
$keyPass = 'changeit'

function Invoke-Keytool {
    param([Parameter(ValueFromRemainingArguments = $true)][string[]]$Args)

    if ($env:JAVA_HOME -and (Test-Path (Join-Path $env:JAVA_HOME 'bin\keytool.exe'))) {
        & (Join-Path $env:JAVA_HOME 'bin\keytool.exe') @Args
        if ($LASTEXITCODE -ne 0) { throw "keytool failed with exit code $LASTEXITCODE" }
        return
    }

    $localKeytool = Get-Command keytool -ErrorAction SilentlyContinue
    if ($localKeytool) {
        & $localKeytool.Source @Args
        if ($LASTEXITCODE -ne 0) { throw "keytool failed with exit code $LASTEXITCODE" }
        return
    }

    # Fallback: run keytool inside ephemeral OpenJDK container
    $pwdPath = (Get-Location).Path
    docker run --rm -v "${pwdPath}:/work" -w /work eclipse-temurin:17-jdk keytool @Args
    if ($LASTEXITCODE -ne 0) { throw "docker keytool failed with exit code $LASTEXITCODE" }
}

New-Item -ItemType Directory -Force -Path $brokersDir | Out-Null
New-Item -ItemType Directory -Force -Path $clientsDir | Out-Null
New-Item -ItemType Directory -Force -Path $tmpDir | Out-Null

Push-Location $tmpDir

if (Test-Path 'ca.key.pem') { Remove-Item 'ca.key.pem' -Force }
if (Test-Path 'ca.cert.pem') { Remove-Item 'ca.cert.pem' -Force }

openssl req -new -x509 -keyout ca.key.pem -out ca.cert.pem -days 3650 -passout pass:$storePass -subj "/CN=kafka-ca"
Copy-Item ca.cert.pem (Join-Path $certBase 'ca.cert.pem') -Force

function New-BrokerCert {
    param([string]$Name)

    $extFile = "$Name.ext"
    @"
subjectAltName=DNS:$Name
extendedKeyUsage=serverAuth,clientAuth
"@ | Set-Content -Path $extFile -Encoding ascii

    Invoke-Keytool -genkeypair -alias $Name -keystore "$Name.keystore.jks" -storepass $storePass -keypass $keyPass -dname "CN=$Name" -keyalg RSA -validity 3650
    Invoke-Keytool -keystore "$Name.keystore.jks" -alias $Name -certreq -file "$Name.csr" -storepass $storePass

    openssl x509 -req -CA ca.cert.pem -CAkey ca.key.pem -in "$Name.csr" -out "$Name-signed.crt" -days 3650 -CAcreateserial -passin pass:$storePass -extfile $extFile

    Invoke-Keytool -keystore "$Name.keystore.jks" -alias CARoot -import -file ca.cert.pem -storepass $storePass -noprompt
    Invoke-Keytool -keystore "$Name.keystore.jks" -alias $Name -import -file "$Name-signed.crt" -storepass $storePass -noprompt

    Invoke-Keytool -keystore "$Name.truststore.jks" -alias CARoot -import -file ca.cert.pem -storepass $storePass -noprompt

    Copy-Item "$Name.keystore.jks" (Join-Path $brokersDir "$Name.keystore.jks") -Force
    Copy-Item "$Name.truststore.jks" (Join-Path $brokersDir "$Name.truststore.jks") -Force
}

function New-ClientCert {
    param([string]$Name)

    Invoke-Keytool -genkeypair -alias $Name -keystore "$Name.keystore.jks" -storepass $storePass -keypass $keyPass -dname "CN=$Name" -keyalg RSA -validity 3650
    Invoke-Keytool -keystore "$Name.keystore.jks" -alias $Name -certreq -file "$Name.csr" -storepass $storePass

    openssl x509 -req -CA ca.cert.pem -CAkey ca.key.pem -in "$Name.csr" -out "$Name-signed.crt" -days 3650 -CAcreateserial -passin pass:$storePass

    Invoke-Keytool -keystore "$Name.keystore.jks" -alias CARoot -import -file ca.cert.pem -storepass $storePass -noprompt
    Invoke-Keytool -keystore "$Name.keystore.jks" -alias $Name -import -file "$Name-signed.crt" -storepass $storePass -noprompt

    Invoke-Keytool -keystore "$Name.truststore.jks" -alias CARoot -import -file ca.cert.pem -storepass $storePass -noprompt

    Invoke-Keytool -importkeystore -srckeystore "$Name.keystore.jks" -srcstorepass $storePass -destkeystore "$Name.p12" -deststoretype PKCS12 -deststorepass $storePass -noprompt
    openssl pkcs12 -in "$Name.p12" -passin pass:$storePass -nodes -nocerts | Out-File "$Name.key.pem" -Encoding ascii
    Invoke-Keytool -exportcert -rfc -keystore "$Name.keystore.jks" -alias $Name -storepass $storePass > "$Name.cert.pem"

    Copy-Item "$Name.keystore.jks" (Join-Path $clientsDir "$Name.keystore.jks") -Force
    Copy-Item "$Name.truststore.jks" (Join-Path $clientsDir "$Name.truststore.jks") -Force
    Copy-Item "$Name.key.pem" (Join-Path $clientsDir "$Name.key.pem") -Force
    Copy-Item "$Name.cert.pem" (Join-Path $clientsDir "$Name.cert.pem") -Force
}

New-BrokerCert -Name 'kafka-1'
New-BrokerCert -Name 'kafka-2'
New-BrokerCert -Name 'kafka-3'

New-ClientCert -Name 'admin'
New-ClientCert -Name 'producer'
New-ClientCert -Name 'consumer'

Pop-Location
Write-Host 'Certificates generated under ./certs'


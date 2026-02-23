# Kafka Message Filter System

Система обработки потоков сообщений с функциями блокировки пользователей и цензуры сообщений на базе Apache Kafka и библиотеки Goka.

## Описание архитектуры

### Компоненты системы

#### 1. **Block Manager (Менеджер блокировок)**
- **Назначение**: Управление списками заблокированных пользователей
- **Топик входа**: `blocked_users`
- **Group Table**: `block-manager-blocked-users-table`
- **Функционал**:
  - Обрабатывает действия блокировки/разблокировки пользователей
  - Хранит персистентный список заблокированных пользователей для каждого пользователя
  - Поддерживает операции: `block` (заблокировать) и `unblock` (разблокировать)

#### 2. **Censor Manager (Менеджер цензуры)**
- **Назначение**: Управление списком запрещённых слов
- **Топик входа**: `censor-actions`
- **Group Table**: `censor-manager-censored-words-table`
- **Функционал**:
  - Обрабатывает действия добавления/удаления запрещённых слов
  - Хранит глобальный персистентный список запрещённых слов
  - Поддерживает операции: `add` (добавить слово) и `remove` (удалить слово)

#### 3. **Message Filter (Фильтр сообщений)**
- **Назначение**: Фильтрация и цензурирование сообщений
- **Топик входа**: `messages`
- **Топик выхода**: `filtered_messages`
- **Lookups**: Использует таблицы Block Manager и Censor Manager
- **Функционал**:
  - Проверяет, не заблокирован ли отправитель получателем
  - Применяет цензуру к содержимому сообщения
  - Отправляет отфильтрованные сообщения получателям

### Потоки данных (Kafka Topics)

| Топик | Описание | Ключ | Значение |
|-------|----------|------|----------|
| `messages` | Входящие сообщения | `from` (отправитель) | Message |
| `filtered_messages` | Отфильтрованные сообщения | `to` (получатель) | Message |
| `block-actions` | Действия блокировки | `blocker_id` | BlockAction |
| `censor-actions` | Действия цензуры | `"global"` | CensorAction |

### Модели данных

#### Message (Сообщение)
```go
type Message struct {
    ID        string // Уникальный ID сообщения
    From      string // Отправитель
    To        string // Получатель
    Content   string // Содержимое сообщения
    Timestamp int64  // Время отправки (Unix timestamp)
}
```

#### BlockAction (Действие блокировки)
```go
type BlockAction struct {
    BlockerID string // ID пользователя, который блокирует
    BlockedID string // ID блокируемого пользователя
    Action    string // "block" или "unblock"
    Timestamp int64  // Время действия
}
```

#### CensorAction (Действие цензуры)
```go
type CensorAction struct {
    Word      string // Запрещённое слово
    Action    string // "add" или "remove"
    Timestamp int64  // Время действия
}
```

#### BlockedUsersList (Список заблокированных)
```go
type BlockedUsersList struct {
    Users map[string]BlockedUser // Карта заблокированных пользователей
}
```

#### CensoredWordsList (Список запрещённых слов)
```go
type CensoredWordsList struct {
    Words map[string]CensoredWord // Карта запрещённых слов
}
```

## Логика работы приложения

### 1. Обработка блокировки пользователей

```
Пользователь A блокирует пользователя B
    ↓
Отправка BlockAction в топик "blocked_users" с ключом A
    ↓
Block Manager обрабатывает действие
    ↓
Обновление персистентного списка заблокированных для пользователя A
    ↓
Список сохраняется в Group Table
```

### 2. Обработка цензуры

```
Администратор добавляет запрещённое слово
    ↓
Отправка CensorAction в топик "censor-actions"
    ↓
Censor Manager обрабатывает действие
    ↓
Обновление глобального списка запрещённых слов
    ↓
Список сохраняется в Group Table
```

### 3. Фильтрация сообщений

```
Сообщение поступает в топик "messages"
    ↓
Message Filter получает сообщение
    ↓
Проверка: заблокирован ли отправитель получателем?
    ├─ ДА → Сообщение отбрасывается
    └─ НЕТ → Продолжение обработки
         ↓
Применение цензуры (замена запрещённых слов на ***)
    ↓
Отправка в топик "filtered_messages"
```

## Инструкция по запуску

### Предварительные требования

- Docker 20.10+ и Docker Compose 2.0+
- Минимум 4GB RAM для Docker
- (Опционально) Go 1.21+ для локальной разработки

### Вариант 1: Docker Compose (Рекомендуется) ⭐

Все приложения запускаются в изолированных контейнерах:

```bash
# 1. Запуск всей системы (Kafka + процессор + консьюмер)
docker-compose up -d

# 2. Просмотр логов процессора
docker-compose logs -f app-processor

# 3. Просмотр логов консьюмера (отфильтрованные сообщения)
docker-compose logs -f app-consumer

# 4. Запуск тестового продюсера (отправка тестовых данных)
docker-compose --profile test up app-producer
```

После запуска:
- Kafka доступен на `localhost:9092`
- Kafka UI доступен на `http://localhost:8080`
- Процессор автоматически запускается и обрабатывает сообщения
- Консьюмер показывает отфильтрованные сообщения

**Полезные команды:**
```bash
# Просмотр всех логов
docker-compose logs -f

# Пересборка после изменения кода
docker-compose up -d --build

# Остановка всех контейнеров
docker-compose down

# Остановка с удалением данных
docker-compose down -v
```

### Вариант 2: Локальная разработка

Для разработки и отладки можно запускать приложения локально:

```bash
# 1. Запуск только Kafka инфраструктуры
docker-compose up -d kafka zookeeper kafka-ui

# Ожидание готовности Kafka (30-60 секунд)

# 2. Установка Go зависимостей
go mod download

# 3. В первом терминале - запуск процессора
go run cmd/processor/main.go

# 4. Во втором терминале - просмотр результатов
go run cmd/consumer/main.go

# 5. В третьем терминале - отправка тестов
go run cmd/producer/main.go
```

### Вариант 3: Make команды

```bash
# Для Docker режима
make docker-up          # Запуск всей системы
make docker-test        # Запуск тестов
make docker-logs        # Просмотр логов
make docker-down        # Остановка

# Для локальной разработки
make setup              # Установка инфраструктуры
make processor          # Запуск процессора
make consumer           # Просмотр результатов
make producer           # Отправка тестов
```

## Инструкция по тестированию

### Автоматическое тестирование в Docker

```bash
# Запуск тестового продюсера в Docker
docker-compose --profile test up app-producer

# Просмотр результатов в логах консьюмера
docker-compose logs -f app-consumer
```

### Автоматическое тестирование локально

```bash
go run cmd/producer/main.go
```

Тестовый сценарий включает:
1. Добавление запрещённых слов
2. Отправка обычных сообщений
3. Отправка сообщений с запрещёнными словами
4. Блокировка пользователей
5. Отправка сообщений от заблокированных пользователей
6. Разблокировка пользователей
7. Удаление запрещённых слов

### Ручное тестирование

#### Создание топиков (если нужно вручную)

```bash
# Подключение к контейнеру Kafka
docker exec -it kafka bash

# Создание топиков
kafka-topics --create --topic messages --bootstrap-server localhost:9093 --partitions 3 --replication-factor 1
kafka-topics --create --topic filtered_messages --bootstrap-server localhost:9093 --partitions 3 --replication-factor 1
kafka-topics --create --topic blocked_users --bootstrap-server localhost:9093 --partitions 3 --replication-factor 1
kafka-topics --create --topic censor-actions --bootstrap-server localhost:9093 --partitions 1 --replication-factor 1

# Просмотр списка топиков
kafka-topics --list --bootstrap-server localhost:9093
```

#### Отправка данных вручную через Kafka Console Producer

```bash
# Добавление запрещённых слов
docker exec -it kafka bash
kafka-console-producer --topic censor-actions --bootstrap-server localhost:9093 --property "parse.key=true" --property "key.separator=:"

# Введите (каждая строка - отдельное сообщение):
global:{"word":"badword","action":"add","timestamp":1708368000}
global:{"word":"spam","action":"add","timestamp":1708368001}
```

```bash
# Блокировка пользователей
kafka-console-producer --topic blocked_users --bootstrap-server localhost:9093 --property "parse.key=true" --property "key.separator=:"

# Введите:
alice:{"blocker_id":"alice","blocked_id":"bob","action":"block","timestamp":1708368100}
```

```bash
# Отправка сообщений
kafka-console-producer --topic messages --bootstrap-server localhost:9093 --property "parse.key=true" --property "key.separator=:"

# Введите:
alice:{"id":"msg1","from":"alice","to":"bob","content":"Hello Bob!","timestamp":1708368200}
charlie:{"id":"msg2","from":"charlie","to":"alice","content":"This is spam message","timestamp":1708368201}
```

#### Просмотр отфильтрованных сообщений

```bash
# Чтение из топика filtered_messages
docker exec -it kafka bash
kafka-console-consumer --topic filtered_messages --bootstrap-server localhost:9093 --from-beginning
```

## Тестовые данные

### Тест 1: Добавление запрещённых слов

**Топик**: `censor-actions`

```json
{"word":"badword","action":"add","timestamp":1708368000}
{"word":"spam","action":"add","timestamp":1708368001}
{"word":"offensive","action":"add","timestamp":1708368002}
```

**Ожидаемый результат**: Слова добавлены в список запрещённых

### Тест 2: Обычные сообщения (без блокировки и цензуры)

**Топик**: `messages`

```json
{"id":"msg1","from":"alice","to":"bob","content":"Hello Bob, how are you?","timestamp":1708368100}
{"id":"msg2","from":"bob","to":"alice","content":"Hi Alice, I'm doing great!","timestamp":1708368101}
```

**Ожидаемый результат**: Сообщения появляются в `filtered_messages` без изменений

### Тест 3: Сообщения с запрещёнными словами

**Топик**: `messages`

```json
{"id":"msg3","from":"charlie","to":"bob","content":"This is a badword message!","timestamp":1708368200}
{"id":"msg4","from":"alice","to":"bob","content":"No spam here, just testing","timestamp":1708368201}
{"id":"msg5","from":"dave","to":"alice","content":"This contains offensive content","timestamp":1708368202}
```

**Ожидаемый результат**: В `filtered_messages`:
- "This is a ******* message!"
- "No **** here, just testing"
- "This contains ********* content"

### Тест 4: Блокировка пользователей

**Топик**: `blocked_users`

```json
{"blocker_id":"bob","blocked_id":"charlie","action":"block","timestamp":1708368300}
{"blocker_id":"alice","blocked_id":"dave","action":"block","timestamp":1708368301}
```

**Ожидаемый результат**: Charlie заблокирован Bob'ом, Dave заблокирован Alice

### Тест 5: Сообщения от заблокированных пользователей

**Топик**: `messages`

```json
{"id":"msg6","from":"charlie","to":"bob","content":"Can you see this Bob?","timestamp":1708368400}
{"id":"msg7","from":"dave","to":"alice","content":"Hi Alice!","timestamp":1708368401}
{"id":"msg8","from":"alice","to":"bob","content":"Hello Bob from Alice","timestamp":1708368402}
```

**Ожидаемый результат**: 
- msg6 НЕ появится в `filtered_messages` (Charlie заблокирован Bob'ом)
- msg7 НЕ появится в `filtered_messages` (Dave заблокирован Alice)
- msg8 ПОЯВИТСЯ в `filtered_messages` (Alice не заблокирована Bob'ом)

### Тест 6: Разблокировка пользователей

**Топик**: `blocked_users`

```json
{"blocker_id":"bob","blocked_id":"charlie","action":"unblock","timestamp":1708368500}
```

**Ожидаемый результат**: Charlie разблокирован Bob'ом

### Тест 7: Сообщение после разблокировки

**Топик**: `messages`

```json
{"id":"msg9","from":"charlie","to":"bob","content":"Thanks for unblocking me!","timestamp":1708368600}
```

**Ожидаемый результат**: Сообщение появляется в `filtered_messages`

### Тест 8: Удаление запрещённого слова

**Топик**: `censor-actions`

```json
{"word":"spam","action":"remove","timestamp":1708368700}
```

**Ожидаемый результат**: Слово "spam" удалено из списка запрещённых

### Тест 9: Сообщение с ранее запрещённым словом

**Топик**: `messages`

```json
{"id":"msg10","from":"alice","to":"bob","content":"This spam word should be visible now","timestamp":1708368800}
```

**Ожидаемый результат**: В `filtered_messages`: "This spam word should be visible now" (без цензуры)

## Мониторинг и отладка

### Kafka UI

Откройте http://localhost:8080 для просмотра:
- Списка топиков
- Сообщений в топиках
- Consumer groups
- Состояния брокеров

### Логи процессора

Процессор выводит подробные логи о каждой операции:
- Обработка блокировок
- Обработка цензуры
- Фильтрация сообщений
- Обнаружение заблокированных сообщений
- Применение цензуры

### Просмотр Group Tables

Group Tables хранятся локально в директории процессора. Для просмотра содержимого используйте специальные инструменты Kafka или логи приложения.

## Остановка системы

```bash
# Остановка приложений (Ctrl+C в каждом терминале)

# Остановка Docker контейнеров
docker-compose down

# Для полной очистки (включая данные)
docker-compose down -v
```

## Структура проекта

```
.
├── cmd/
│   ├── processor/       # Основное приложение (процессор)
│   │   └── main.go
│   ├── producer/        # Тестовый продюсер
│   │   └── main.go
│   └── consumer/        # Консьюмер для просмотра результатов
│       └── main.go
├── models/              # Модели данных и кодеки
│   └── message.go
├── processors/          # Логика обработки
│   ├── block_manager.go
│   ├── censor_manager.go
│   └── message_filter.go
├── docker-compose.yml   # Конфигурация инфраструктуры
├── go.mod              # Зависимости Go
└── README.md           # Документация
```

## Технологии

- **Go 1.21+**: Язык программирования
- **Apache Kafka**: Брокер сообщений
- **Goka**: Библиотека для stream processing
- **Sarama**: Kafka клиент для Go
- **Docker & Docker Compose**: Контейнеризация

## Возможные улучшения

1. Добавление метрик и мониторинга (Prometheus, Grafana)
2. Реализация REST API для управления блокировками и цензурой
3. Поддержка регулярных выражений для цензуры
4. Добавление уровней строгости цензуры
5. Реализация временных блокировок
6. Добавление unit и integration тестов
7. Поддержка множественных партиций для масштабируемости
8. Реализация истории блокировок

## Лицензия

Учебный проект для Yandex.Practicum

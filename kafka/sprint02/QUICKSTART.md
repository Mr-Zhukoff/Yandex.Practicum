# Быстрый старт

## Минимальная инструкция для запуска

### Вариант 1: Docker Compose (Рекомендуется) ⭐

Все приложения запускаются в изолированных контейнерах:

```bash
# 1. Запуск всей системы (Kafka + процессор + консьюмер)
docker-compose up -d

# 2. Просмотр логов процессора
docker-compose logs -f app-processor

# 3. Просмотр логов консьюмера
docker-compose logs -f app-consumer

# 4. Запуск тестового продюсера (отправка тестовых данных)
docker-compose --profile test up app-producer

# Или с rebuild если изменили код
docker-compose --profile test up --build app-producer
```

### Вариант 2: Локальный запуск Go (для разработки)

```bash
# 1. Запуск только Kafka инфраструктуры
docker-compose up -d kafka zookeeper kafka-ui

# 2. В первом терминале - запуск процессора
go run cmd/processor/main.go

# 3. Во втором терминале - просмотр результатов
go run cmd/consumer/main.go

# 4. В третьем терминале - отправка тестов
go run cmd/producer/main.go
```

### Вариант 3: Автоматический запуск (с Make)

```bash
# Для Docker режима
make docker-up          # Запуск всей системы в Docker
make docker-test        # Запуск тестового продюсера
make docker-logs        # Просмотр логов
make docker-down        # Остановка

# Для локальной разработки
make setup              # Установка и запуск инфраструктуры
make processor          # Запуск процессора
make consumer           # Просмотр результатов
make producer           # Отправка тестовых данных
```

## Полезные команды Docker

```bash
# Пересборка и запуск (после изменения кода)
docker-compose up -d --build

# Просмотр статуса контейнеров
docker-compose ps

# Просмотр логов всех сервисов
docker-compose logs -f

# Просмотр логов конкретного сервиса
docker-compose logs -f app-processor
docker-compose logs -f app-consumer
docker-compose logs -f kafka

# Остановка всех контейнеров
docker-compose down

# Остановка с удалением данных
docker-compose down -v

# Перезапуск процессора
docker-compose restart app-processor
```

## Что вы увидите

После запуска системы:

1. **Kafka UI** доступен на http://localhost:8080
   - Просмотр топиков
   - Сообщения в реальном времени
   - Статус consumer groups

2. **Логи процессора** (`docker-compose logs -f app-processor`):
   - Обработка блокировок
   - Применение цензуры
   - Фильтрация сообщений

3. **Логи консьюмера** (`docker-compose logs -f app-consumer`):
   - Отфильтрованные сообщения
   - Только разрешённые сообщения с цензурой

4. **Тестовый продюсер** демонстрирует:
   - ✅ Цензуру сообщений - запрещённые слова → `***`
   - ✅ Блокировку пользователей - блокированные не доходят
   - ✅ Разблокировку - восстановление доставки
   - ✅ Динамическое управление списками

## Архитектура контейнеров

```
┌─────────────────┐
│   zookeeper     │
└────────┬────────┘
         │
┌────────▼────────┐
│     kafka       │◄────┐
└────────┬────────┘     │
         │              │
    ┌────▼───────┐      │
    │  kafka-ui  │      │
    └────────────┘      │
                        │
    ┌───────────────────┼───────────────┐
    │                   │               │
┌───▼──────────┐  ┌────▼────────┐  ┌──▼──────────┐
│app-processor │  │app-consumer │  │app-producer │
│(фильтрация)  │  │(просмотр)   │  │(тесты)      │
└──────────────┘  └─────────────┘  └─────────────┘
```

## Структура проекта

```
sprint02/
├── cmd/
│   ├── processor/main.go    ← Основной процессор
│   ├── producer/main.go     ← Тестовые данные
│   └── consumer/main.go     ← Просмотр результатов
├── models/message.go        ← Модели данных
├── processors/              ← Логика обработки
│   ├── block_manager.go     ← Блокировки
│   ├── censor_manager.go    ← Цензура
│   └── message_filter.go    ← Фильтрация
├── Dockerfile              ← Сборка Go приложений
├── docker-compose.yml      ← Оркестрация контейнеров
├── README.md              ← Полная документация
└── Makefile               ← Команды запуска
```

## Тестирование

### Автоматические тесты в Docker

```bash
# Запуск тестового продюсера
docker-compose --profile test up app-producer

# Просмотр результатов в логах
docker-compose logs -f app-consumer
```

### Проверка работы

1. Откройте http://localhost:8080 (Kafka UI)
2. Перейдите в Topics → filtered_messages
3. Увидите только отфильтрованные сообщения
4. Запрещённые слова заменены на `***`
5. Сообщения от заблокированных отсутствуют

## Остановка

```bash
# Остановка контейнеров
docker-compose down

# Остановка с удалением данных
docker-compose down -v

# Остановка только приложений (Kafka продолжит работать)
docker-compose stop app-processor app-consumer
```

## Требования

- Docker 20.10+
- Docker Compose 2.0+
- 4GB RAM для Docker
- (Опционально) Go 1.21+ для локальной разработки

## Помощь

- Полная документация: [`README.md`](README.md)
- Ручное тестирование: [`scripts/test-manual.md`](scripts/test-manual.md)
- Обзор проекта: [`PROJECT_SUMMARY.md`](PROJECT_SUMMARY.md)

## Troubleshooting

**Проблема**: Контейнеры не запускаются
```bash
docker-compose down -v
docker-compose up -d --build
```

**Проблема**: Нет сообщений в filtered_messages
```bash
# Проверьте логи процессора
docker-compose logs app-processor

# Убедитесь что processor запущен
docker-compose ps
```

**Проблема**: Ошибка подключения к Kafka
```bash
# Подождите пока Kafka полностью запустится (30-60 сек)
docker-compose logs kafka
```

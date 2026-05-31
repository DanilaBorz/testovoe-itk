# Wallet Service

REST-сервис для управления кошельками с поддержкой конкурентных операций (1000+ RPS).

## 📋 Содержание

- [Функциональные требования](#-функциональные-требования)
- [Стек технологий](#-стек-технологий)
- [Архитектура](#-архитектура)
- [Конкурентность](#-конкурентность)
- [Быстрый старт](#-быстрый-старт)
- [API](#-api)
- [Тестирование](#-тестирование)
- [Makefile](#-makefile)
- [Что сделано сверх ТЗ](#-что-сделано-сверх-тз)
- [Переменные окружения](#-переменные-окружения)

---

## 📋 Функциональные требования

### POST `/api/v1/wallet`
Выполнить операцию пополнения или снятия средств с кошелька.

```json
{
  "walletId": "550e8400-e29b-41d4-a716-446655440000",
  "operationType": "DEPOSIT",
  "amount": 1000
}
```

**Ответ:**
```json
{
  "walletId": "550e8400-e29b-41d4-a716-446655440000",
  "balance": 1000
}
```

### GET `/api/v1/wallets/{WALLET_UUID}`
Получить текущий баланс кошелька.

**Ответ:**
```json
{
  "walletId": "550e8400-e29b-41d4-a716-446655440000",
  "balance": 1000
}
```

---

## 🛠 Стек технологий

| Компонент | Технология |
|-----------|-----------|
| **Язык** | Go 1.22 |
| **HTTP-роутер** | [chi](https://github.com/go-chi/chi) — лёгкий, производительный, idiomatic |
| **База данных** | PostgreSQL 16 |
| **Драйвер БД** | [pgx v5](https://github.com/jackc/pgx) — пул соединений, нативная поддержка PostgreSQL |
| **Логирование** | [zap](https://github.com/uber-go/zap) — структурированное логирование |
| **Контейнеризация** | Docker + docker-compose |
| **Тестирование** | testify + gomock (go.uber.org/mock) |
| **Конфигурация** | godotenv + переменные окружения |

---

## 🏗 Архитектура

Проект построен по принципу **чистой архитектуры** с разделением на слои:

```
┌─────────────────────────────────────────────────────────┐
│                    HTTP (chi router)                     │
│  POST /api/v1/wallet  │  GET /api/v1/wallets/{id}       │
└───────────────────────┬─────────────────────────────────┘
                        │
┌───────────────────────▼─────────────────────────────────┐
│                    Handler (handler/)                     │
│              Валидация запроса, формирование ответа       │
│              Зависит только от: service.WalletService     │
│              и apperrors                                  │
└───────────────────────┬─────────────────────────────────┘
                        │
┌───────────────────────▼─────────────────────────────────┐
│                    Service (service/)                     │
│              Бизнес-логика: DEPOSIT → +amount             │
│              WITHDRAW → -amount                           │
│              Определяет: WalletRepository (интерфейс)     │
└───────────────────────┬─────────────────────────────────┘
                        │
┌───────────────────────▼─────────────────────────────────┐
│                  Repository (repository/)                 │
│              Работа с PostgreSQL, транзакции,             │
│              блокировка строк SELECT FOR UPDATE           │
│              Реализует: service.WalletRepository          │
└─────────────────────────────────────────────────────────┘
```

### Цепочка зависимостей

```
handler → service.WalletService (интерфейс)
service → WalletRepository (свой интерфейс)
repository → WalletRepo (структура, не импортирует service)
```

Каждый слой зависит только от **интерфейса** нижележащего слоя. Это позволяет легко тестировать каждый слой изолированно через моки.

---

## ⚡ Конкурентность

**Проблема:** 1000 RPS на один кошелёк — конкурентные запросы могут привести к race condition и потере данных.

**Решение:** `SELECT ... FOR UPDATE` в транзакции PostgreSQL.

```
Запрос 1 (DEPOSIT 100)        Запрос 2 (WITHDRAW 50)
       │                            │
       │ BEGIN                      │
       │ SELECT ... FOR UPDATE      │
       │ (блокирует строку)         │
       │                            │ (ждёт освобождения строки)
       │ UPDATE balance = balance+100│
       │ COMMIT                     │
       │ (снимает блокировку)       │
       │                     ┌──────┤
       │                     │ BEGIN│
       │                     │ SELECT ... FOR UPDATE
       │                     │ (получает актуальные данные)
       │                     │ UPDATE balance = balance-50
       │                     │ COMMIT
       ▼                            ▼
Баланс: 100 → 200            Баланс: 200 → 150
```

**Результат:** все операции выполняются атомарно, ни один запрос не теряется, 50Х ошибок нет.

---

## 🚀 Быстрый старт

### 1. Запуск приложения

```bash
docker-compose up --build
```

Сервер будет доступен на `http://localhost:8080`.

### 2. Примеры запросов

**Пополнить кошелёк:**
```bash
curl -X POST http://localhost:8080/api/v1/wallet \
  -H "Content-Type: application/json" \
  -d '{"walletId":"550e8400-e29b-41d4-a716-446655440000","operationType":"DEPOSIT","amount":1000}'
```

**Снять средства:**
```bash
curl -X POST http://localhost:8080/api/v1/wallet \
  -H "Content-Type: application/json" \
  -d '{"walletId":"550e8400-e29b-41d4-a716-446655440000","operationType":"WITHDRAW","amount":500}'
```

**Получить баланс:**
```bash
curl http://localhost:8080/api/v1/wallets/550e8400-e29b-41d4-a716-446655440000
```

---

## 🧪 Тестирование

### Unit-тесты (без БД)

```bash
make test-unit
```

Проверяют handler и service через моки. Покрывают: успешные операции, валидацию, ошибки, неверные методы.

### Интеграционные тесты (с реальной БД)

```bash
# 1. Поднять тестовую PostgreSQL
make db-test-up

# 2. Запустить интеграционные тесты
make test-integration

# 3. Остановить БД
make db-test-down
```

Проверяют repository: создание кошелька, DEPOSIT, WITHDRAW, недостаточно средств, **100 конкурентных операций**.

### Покрытие

```bash
# Unit-тесты с покрытием
make coverage-unit

# Все тесты + HTML-отчёт
make db-test-up && make coverage-html
# открыть coverage/coverage.html в браузере
```

---

## 📦 Makefile

```bash
make help        # Показать краткую справку по целям
make info        # Показать подробное описание всех целей
make build       # Собрать бинарник
make run         # Запустить через docker-compose
make test        # Unit-тесты
make test-all    # Все тесты (unit + integration)
make coverage    # Все тесты + объединённый отчёт о покрытии
make coverage-html # Все тесты + HTML-отчёт
make generate    # Сгенерировать моки
make lint        # Запустить golangci-lint
make clean       # Очистить артефакты
```

---

## ✨ Что сделано сверх ТЗ

| Улучшение | Описание |
|-----------|----------|
| **Чистая архитектура** | Handler → Service → Repository через интерфейсы |
| **Структурированное логирование** | Zap logger с уровнями info/warn/error и контекстными полями |
| **Отдельный пакет ошибок** | `internal/errors` — handler не зависит от repository |
| **Graceful shutdown** | Корректное завершение по SIGINT/SIGTERM |
| **Mockgen + gomock** | Автоматическая генерация моков через `make generate` |
| **Makefile** | Цели для сборки, тестов, покрытия, линтинга |
| **Race detector** | Все тесты запускаются с `-race` |
| **Покрытие кода** | `make coverage-html` — HTML-отчёт в браузере |
| **Валидация конфига** | Проверка обязательных переменных окружения при старте |
| **Интеграционный тест на конкурентность** | 100 горутин на один кошелёк — проверка `SELECT FOR UPDATE` |
| **Docker-оптимизация** | Многоступенчатая сборка (alpine), healthcheck для БД |

---

## 🔧 Переменные окружения

Все переменные считываются из файла `config.env`:

```env
# Server
SERVER_PORT=8080

# Database
DB_HOST=postgres
DB_PORT=5432
DB_USER=wallet_user
DB_PASSWORD=wallet_pass
DB_NAME=wallet_db
DB_SSLMODE=disable

# Pool
DB_MAX_CONNS=50
DB_MIN_CONNS=10

# Logging
LOG_LEVEL=info
```

---

## 📁 Структура проекта

```
├── cmd/server/main.go              # Точка входа
├── internal/
│   ├── config/config.go            # Конфигурация из config.env
│   ├── errors/errors.go            # Общие ошибки (wallet not found, insufficient funds)
│   ├── model/wallet.go             # Модели данных
│   ├── mocks/
│   │   ├── wallet_repository.go    # Сгенерированный мок репозитория
│   │   └── wallet_service.go       # Сгенерированный мок сервиса
│   ├── repository/
│   │   ├── wallet.go               # Слой данных (PostgreSQL + SELECT FOR UPDATE)
│   │   └── wallet_integration_test.go
│   ├── service/
│   │   ├── wallet.go               # Бизнес-логика + интерфейсы
│   │   └── wallet_test.go
│   ├── handler/
│   │   ├── wallet.go               # HTTP-обработчики
│   │   └── wallet_test.go
│   └── server/server.go            # HTTP-сервер + graceful shutdown
├── migrations/001_init.sql         # Создание таблицы wallets
├── Dockerfile                      # Многоступенчатая сборка
├── docker-compose.yml              # app + postgres
├── docker-compose.test.yml         # Тестовая PostgreSQL
├── config.env                      # Переменные окружения
├── Makefile                        # Цели для сборки, тестов и т.д.
└── go.mod
```
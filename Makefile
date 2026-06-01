.PHONY: help info build run test-unit test-integration test-all lint clean generate \
        db-up db-down db-test-up db-test-down docker-up docker-down \
        coverage

APP_NAME := wallet-app
BUILD_DIR := ./bin
COVERAGE_DIR := ./coverage
MOCKGEN_VERSION := v0.6.0
MOCKGEN := $(shell go env GOPATH)/bin/mockgen
SERVICE_PACKAGE := github.com/DanilaBorz/testovoe-itk/internal/service

help: ## Показать краткую справку
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

info: ## Показать подробное описание целей
	@echo "═══════════════════════════════════════════════════════════════"
	@echo "  $(APP_NAME) — цели Makefile"
	@echo "═══════════════════════════════════════════════════════════════"
	@echo ""
	@echo "  🔧 СБОРКА И ЗАПУСК"
	@echo "  ────────────────────────────────────────────────────────────"
	@echo "  build            Собрать бинарник в ./bin/$(APP_NAME)"
	@echo "  run              Запустить приложение через docker compose"
	@echo "  docker-up        Запустить все сервисы в фоне"
	@echo "  docker-down      Остановить все сервисы"
	@echo ""
	@echo "  🧪 ТЕСТЫ"
	@echo "  ────────────────────────────────────────────────────────────"
	@echo "  test-unit        Запустить модульные тесты (обработчики + сервисы)"
	@echo "  test-integration Запустить интеграционные тесты (тестовая БД поднимется автоматически)"
	@echo "  test-all         Сгенерировать моки и запустить все тесты"
	@echo ""
	@echo "  📊 ПОКРЫТИЕ"
	@echo "  ────────────────────────────────────────────────────────────"
	@echo "  coverage         Все тесты + суммарный отчёт о покрытии"
	@echo ""
	@echo "  🛠  УТИЛИТЫ"
	@echo "  ────────────────────────────────────────────────────────────"
	@echo "  generate         Установить mockgen и сгенерировать моки"
	@echo "  lint             Запустить golangci-lint"
	@echo "  clean            Очистить артефакты сборки и покрытия"
	@echo ""
	@echo "  🗄  БАЗА ДАННЫХ"
	@echo "  ────────────────────────────────────────────────────────────"
	@echo "  db-up            Запустить PostgreSQL (основной)"
	@echo "  db-down          Остановить PostgreSQL"
	@echo "  db-test-up       Запустить PostgreSQL для тестов (порт 5433)"
	@echo "  db-test-down     Остановить тестовую PostgreSQL"
	@echo ""

build: ## Собрать бинарный файл приложения
	@echo "Сборка $(APP_NAME)..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build -o $(BUILD_DIR)/$(APP_NAME) ./cmd/server
	@echo "Готово: $(BUILD_DIR)/$(APP_NAME)"

run: ## Запустить приложение через docker compose
	docker compose up --build

docker-up: ## Запустить все сервисы через docker compose
	docker compose up -d --build

docker-down: ## Остановить все сервисы
	docker compose down

generate: ## Установить mockgen и сгенерировать моки из интерфейсов
	@echo "Установка mockgen..."
	@if [ ! -x "$(MOCKGEN)" ] || [ "$$($(MOCKGEN) -version)" != "$(MOCKGEN_VERSION)" ]; then \
		go install go.uber.org/mock/mockgen@$(MOCKGEN_VERSION); \
	fi
	@echo "Генерация моков..."
	$(MOCKGEN) -destination=internal/mocks/wallet_service.go -package=mocks $(SERVICE_PACKAGE) WalletService
	$(MOCKGEN) -destination=internal/mocks/wallet_repository.go -package=mocks $(SERVICE_PACKAGE) WalletRepository
	@echo "Готово"

test-unit: ## Запустить модульные тесты (обработчики + сервисы)
	@echo "Запуск модульных тестов..."
	go test ./internal/handler/... ./internal/service/... -v -count=1 -race

test-integration: db-test-up ## Запустить интеграционные тесты (поднимает тестовую БД)
	@set -e; \
		trap 'status=$$?; docker compose --profile test stop postgres-test || true; docker compose --profile test rm -f postgres-test || true; exit $$status' EXIT; \
		echo "Запуск интеграционных тестов..."; \
		go test ./internal/repository/... -v -count=1 -race -tags=integration

test-all: generate test-unit test-integration ## Сгенерировать моки и запустить все тесты

coverage: generate db-test-up ## Запустить все тесты и вывести суммарное покрытие
	@set -e; \
		trap 'status=$$?; docker compose --profile test stop postgres-test || true; docker compose --profile test rm -f postgres-test || true; exit $$status' EXIT; \
		echo "Запуск модульных тестов с покрытием..."; \
		mkdir -p $(COVERAGE_DIR); \
		go test ./internal/handler/... ./internal/service/... -count=1 -race \
			-coverprofile=$(COVERAGE_DIR)/unit.out -covermode=atomic; \
		echo "Запуск интеграционных тестов с покрытием..."; \
		go test ./internal/repository/... -count=1 -race -tags=integration \
			-coverprofile=$(COVERAGE_DIR)/integration.out -covermode=atomic; \
		echo "---"; \
		echo "Объединение профилей покрытия..."; \
		head -1 $(COVERAGE_DIR)/unit.out > $(COVERAGE_DIR)/combined.out; \
		tail -n +2 $(COVERAGE_DIR)/unit.out >> $(COVERAGE_DIR)/combined.out; \
		tail -n +2 $(COVERAGE_DIR)/integration.out >> $(COVERAGE_DIR)/combined.out; \
		go tool cover -func=$(COVERAGE_DIR)/combined.out | grep total

lint: ## Запустить golangci-lint
	@echo "Запуск линтера..."
	golangci-lint run ./...

clean: ## Очистить артефакты сборки и покрытия
	@echo "Очистка..."
	@rm -rf $(BUILD_DIR) $(COVERAGE_DIR)
	@echo "Готово"

db-up: ## Запустить PostgreSQL
	docker compose up -d postgres

db-down: ## Остановить PostgreSQL
	docker compose stop postgres

db-test-up: ## Запустить тестовую PostgreSQL на порту 5433
	docker compose --profile test up -d --wait postgres-test

db-test-down: ## Остановить тестовую PostgreSQL
	docker compose --profile test stop postgres-test
	docker compose --profile test rm -f postgres-test

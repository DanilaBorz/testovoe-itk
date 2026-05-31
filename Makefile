.PHONY: help info build run test test-unit test-integration test-all lint clean generate \
        db-up db-down db-test-up db-test-down docker-up docker-down \
        coverage coverage-unit coverage-integration coverage-html

APP_NAME := wallet-app
BUILD_DIR := ./bin
COVERAGE_DIR := ./coverage
MOCKGEN := $(GOPATH)/bin/mockgen

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

info: ## Show detailed description of all targets
	@echo "═══════════════════════════════════════════════════════════════"
	@echo "  $(APP_NAME) — Makefile targets"
	@echo "═══════════════════════════════════════════════════════════════"
	@echo ""
	@echo "  🔧 BUILD & RUN"
	@echo "  ────────────────────────────────────────────────────────────"
	@echo "  build            Собрать бинарник в ./bin/$(APP_NAME)"
	@echo "  run              Запустить приложение через docker compose"
	@echo "  docker-up        Запустить все сервисы в фоне"
	@echo "  docker-down      Остановить все сервисы"
	@echo ""
	@echo "  🧪 TESTS"
	@echo "  ────────────────────────────────────────────────────────────"
	@echo "  test             Запустить unit-тесты (по умолчанию)"
	@echo "  test-unit        Запустить unit-тесты (handler + service)"
	@echo "  test-integration Запустить интеграционные тесты (нужна БД)"
	@echo "  test-all         Сгенерировать моки и запустить все тесты"
	@echo ""
	@echo "  📊 COVERAGE"
	@echo "  ────────────────────────────────────────────────────────────"
	@echo "  coverage-unit    Unit-тесты с отчётом о покрытии"
	@echo "  coverage-int     Интеграционные тесты с отчётом о покрытии"
	@echo "  coverage         Все тесты + объединённый отчёт"
	@echo "  coverage-html    Все тесты + HTML-отчёт (открыть в браузере)"
	@echo ""
	@echo "  🛠  UTILITIES"
	@echo "  ────────────────────────────────────────────────────────────"
	@echo "  generate         Установить mockgen и сгенерировать моки"
	@echo "  lint             Запустить golangci-lint"
	@echo "  clean            Очистить артефакты сборки и покрытия"
	@echo ""
	@echo "  🗄  DATABASE"
	@echo "  ────────────────────────────────────────────────────────────"
	@echo "  db-up            Запустить PostgreSQL (основной)"
	@echo "  db-down          Остановить PostgreSQL"
	@echo "  db-test-up       Запустить PostgreSQL для тестов (порт 5433)"
	@echo "  db-test-down     Остановить тестовую PostgreSQL"
	@echo ""

build: ## Build the application binary
	@echo "Building $(APP_NAME)..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build -o $(BUILD_DIR)/$(APP_NAME) ./cmd/server
	@echo "Done: $(BUILD_DIR)/$(APP_NAME)"

run: ## Run the application with docker compose
	docker compose up --build

docker-up: ## Start all services with docker compose
	docker compose up -d --build

docker-down: ## Stop all services
	docker compose down

generate: ## Install mockgen and generate mocks from interfaces
	@echo "Installing mockgen..."
	@go install go.uber.org/mock/mockgen@latest
	@echo "Generating mocks..."
	$(MOCKGEN) -source=internal/service/wallet.go -destination=internal/mocks/wallet_service.go -package=mocks WalletService
	$(MOCKGEN) -source=internal/service/wallet.go -destination=internal/mocks/wallet_repository.go -package=mocks WalletRepository
	@echo "Done"

test-unit: ## Run unit tests (handler + service)
	@echo "Running unit tests..."
	go test ./internal/handler/... ./internal/service/... -v -count=1 -race

test-integration: ## Run integration tests (requires test database)
	@echo "Running integration tests..."
	go test ./internal/repository/... -v -count=1 -race -tags=integration

test-all: generate test-unit test-integration ## Run all tests (generate mocks first)

test: test-unit ## Alias for unit tests (default)

coverage-unit: generate ## Run unit tests with coverage report
	@echo "Running unit tests with coverage..."
	@mkdir -p $(COVERAGE_DIR)
	go test ./internal/handler/... ./internal/service/... -count=1 -race \
		-coverprofile=$(COVERAGE_DIR)/unit.out -covermode=atomic
	@go tool cover -func=$(COVERAGE_DIR)/unit.out | grep total

coverage-integration: ## Run integration tests with coverage report
	@echo "Running integration tests with coverage..."
	@mkdir -p $(COVERAGE_DIR)
	go test ./internal/repository/... -count=1 -race -tags=integration \
		-coverprofile=$(COVERAGE_DIR)/integration.out -covermode=atomic
	@go tool cover -func=$(COVERAGE_DIR)/integration.out | grep total

coverage: coverage-unit coverage-integration ## Run all tests with combined coverage report
	@echo "---"
	@echo "Merging coverage profiles..."
	@head -1 $(COVERAGE_DIR)/unit.out > $(COVERAGE_DIR)/combined.out
	@tail -n +2 $(COVERAGE_DIR)/unit.out >> $(COVERAGE_DIR)/combined.out
	@tail -n +2 $(COVERAGE_DIR)/integration.out >> $(COVERAGE_DIR)/combined.out
	@go tool cover -func=$(COVERAGE_DIR)/combined.out | grep total

coverage-html: coverage ## Generate HTML coverage report and open in browser
	@go tool cover -html=$(COVERAGE_DIR)/combined.out -o $(COVERAGE_DIR)/coverage.html
	@echo "HTML report: $(COVERAGE_DIR)/coverage.html"

lint: ## Run golangci-lint
	@echo "Running linter..."
	golangci-lint run ./...

clean: ## Clean build and coverage artifacts
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR) $(COVERAGE_DIR)
	@echo "Done"

db-up: ## Start PostgreSQL
	docker compose up -d postgres

db-down: ## Stop PostgreSQL
	docker compose stop postgres

db-test-up: ## Start test PostgreSQL on port 5433
	docker compose -f docker compose.test.yml up -d

db-test-down: ## Stop test PostgreSQL
	docker compose -f docker compose.test.yml down
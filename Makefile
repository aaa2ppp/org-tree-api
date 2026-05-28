BIN_DIR ?= ./bin
BUILD_FLAGS ?= 
GOEXE := $(shell go env GOEXE)


LOG_LEVEL ?= INFO

DOCKER_COMPOSE := docker compose
APP_SERVICE    := server
DB_SERVICE     := db
K6_SERVICE     := k6 grafana influxdb
EXTAPI_SERVICE := extapi

USE_EXTERNAL_DB   ?= no
DB_UP_NEEDED      := $(if $(filter yes,$(USE_EXTERNAL_DB)),,db-up)
DB_ADDR           := $(if $(filter yes,$(USE_EXTERNAL_DB)),,localhost:5432)
DB_CHECK_TIMEOUT  := 30
DB_CHECK_INTERVAL := 2
WAIT_DB_READY     := sh scripts/wait-db-ready.sh
MIGRATE           := sh scripts/migrate.sh


all:
	echo "now nothing"

FORCE:

# Правило для подготовки зависимостей
deps: ## update deps
	go mod tidy


# Находим все поддиректории в cmd, которые потенциально могут быть бинарниками
CMDS := $(wildcard cmd/*)

# Генерируем список целей для бинарников
BINARIES := $(patsubst cmd/%,$(BIN_DIR)/%,$(CMDS))

# Шаблонное правило для сборки любого бинарника
$(BIN_DIR)/%: FORCE
	@mkdir -p $(@D)
	go build $(BUILD_FLAGS) -o $@$(GOEXE) ./cmd/$(notdir $@)

build: $(BINARIES) ## build all binaries

clean: ## remove temporary and binary files
	-rm -rf $(BIN_DIR) $(TMP_DIR)


check-goose: ## Check goose
	@which goose 2>/dev/null || go install github.com/pressly/goose/v3/cmd/goose@latest

migrate-up: $(DB_UP_NEEDED) ## Apply all migrations
	DB_ADDR=$(DB_ADDR) $(MIGRATE) up

migrate-down: $(DB_UP_NEEDED) ## Rollback last migration
	DB_ADDR=$(DB_ADDR) $(MIGRATE) down 1

migrate-status: $(DB_UP_NEEDED) ## Show migration status
	DB_ADDR=$(DB_ADDR) $(MIGRATE) status


db-up: ## Start only database
	@if [ -z "$$($(DOCKER_COMPOSE) ps -q $(DB_SERVICE))" ]; then \
		$(DOCKER_COMPOSE) up -d $(DB_SERVICE) && \
		DOCKER_COMPOSE='$(DOCKER_COMPOSE)' $(WAIT_DB_READY); \
	fi

db-down: ## Stop database
	$(DOCKER_COMPOSE) down $(DB_SERVICE)

db-down-volumes: ## Stop database and remove database volumes
	$(DOCKER_COMPOSE) down -v $(DB_SERVICE)


help: ## Display this help
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z0-9_-]+:.*?## / {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)
